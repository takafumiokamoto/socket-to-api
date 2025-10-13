package worker

import (
	"context"
	"sync"
	"time"

	"github.com/okamoto/socket-to-api/internal/config"
	"github.com/okamoto/socket-to-api/internal/models"
	"go.uber.org/zap"
)

// Pool represents a worker pool for processing requests
type Pool struct {
	config    *config.WorkerConfig
	processor *Processor
	jobs      chan *models.ProcessingJob
	results   chan *models.ProcessingResult
	logger    *zap.Logger
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewPool creates a new worker pool
func NewPool(cfg *config.WorkerConfig, processor *Processor, logger *zap.Logger) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		config:    cfg,
		processor: processor,
		jobs:      make(chan *models.ProcessingJob, cfg.QueueSize),
		results:   make(chan *models.ProcessingResult, cfg.QueueSize),
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the worker pool
func (p *Pool) Start() {
	p.logger.Info("starting worker pool", zap.Int("pool_size", p.config.PoolSize))

	// Start workers
	for i := 0; i < p.config.PoolSize; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	// Start result handler
	p.wg.Add(1)
	go p.resultHandler()
}

// worker processes jobs from the job queue
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	p.logger.Debug("worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Debug("worker stopping", zap.Int("worker_id", id))
			return

		case job, ok := <-p.jobs:
			if !ok {
				p.logger.Debug("job channel closed", zap.Int("worker_id", id))
				return
			}

			p.processJob(id, job)
		}
	}
}

// processJob processes a single job
func (p *Pool) processJob(workerID int, job *models.ProcessingJob) {
	startTime := time.Now()

	p.logger.Debug("processing job",
		zap.Int("worker_id", workerID),
		zap.Int64("request_id", job.Request.ID),
		zap.Int("retry_count", job.RetryCount))

	// Create a context with timeout for this job
	ctx, cancel := context.WithTimeout(p.ctx, p.config.ProcessTimeout)
	defer cancel()

	// Process the request
	result := p.processor.Process(ctx, job.Request)
	result.ProcessedAt = time.Now()

	duration := time.Since(startTime)
	if result.Success {
		p.logger.Info("job processed successfully",
			zap.Int("worker_id", workerID),
			zap.Int64("request_id", job.Request.ID),
			zap.Duration("duration", duration))
	} else {
		p.logger.Error("job processing failed",
			zap.Int("worker_id", workerID),
			zap.Int64("request_id", job.Request.ID),
			zap.Duration("duration", duration),
			zap.Error(result.Error))
	}

	// Send result to result handler
	select {
	case p.results <- result:
	case <-p.ctx.Done():
		return
	}
}

// resultHandler handles processing results
func (p *Pool) resultHandler() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return

		case result, ok := <-p.results:
			if !ok {
				return
			}

			p.handleResult(result)
		}
	}
}

// handleResult handles a single processing result
func (p *Pool) handleResult(result *models.ProcessingResult) {
	// The result handling is delegated to the processor
	// which will update the database and send responses to clients
	p.logger.Debug("result handled",
		zap.Int64("request_id", result.RequestID),
		zap.Bool("success", result.Success))
}

// Submit submits a job to the worker pool
func (p *Pool) Submit(job *models.ProcessingJob) error {
	select {
	case p.jobs <- job:
		p.logger.Debug("job submitted",
			zap.Int64("request_id", job.Request.ID))
		return nil

	case <-p.ctx.Done():
		return context.Canceled

	default:
		// Queue is full
		p.logger.Warn("job queue full, rejecting job",
			zap.Int64("request_id", job.Request.ID))
		return ErrQueueFull
	}
}

// Stop gracefully stops the worker pool
func (p *Pool) Stop() {
	p.logger.Info("stopping worker pool")

	// Signal shutdown
	p.cancel()

	// Close job channel to signal workers to stop
	close(p.jobs)

	// Wait for all workers to finish
	p.wg.Wait()

	// Close results channel
	close(p.results)

	p.logger.Info("worker pool stopped")
}

// Stats returns statistics about the worker pool
func (p *Pool) Stats() PoolStats {
	return PoolStats{
		PoolSize:      p.config.PoolSize,
		QueueSize:     p.config.QueueSize,
		JobsInQueue:   len(p.jobs),
		ResultsInQueue: len(p.results),
	}
}

// PoolStats represents worker pool statistics
type PoolStats struct {
	PoolSize       int
	QueueSize      int
	JobsInQueue    int
	ResultsInQueue int
}

// Custom errors
var (
	ErrQueueFull = &PoolError{Message: "job queue is full"}
)

// PoolError represents a worker pool error
type PoolError struct {
	Message string
}

func (e *PoolError) Error() string {
	return e.Message
}
