// Package event provides functionality to process telemetry events and forward them to registered providers.If thIf
package event

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/shaharia-lab/telemetry-forwarder/internal/provider"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
)

// Processor is responsible for processing telemetry events and forwarding them to registered providers.
type Processor struct {
	providerRegistry *provider.Registry
	eventChan        chan types.OTelEvent
	done             chan struct{}
	wg               sync.WaitGroup
}

// NewEventProcessor creates a new event processor with the given provider registry and buffer size.
func NewEventProcessor(providerRegistry *provider.Registry, bufferSize int) *Processor {
	return &Processor{
		providerRegistry: providerRegistry,
		eventChan:        make(chan types.OTelEvent, bufferSize),
		done:             make(chan struct{}),
	}
}

// Start begins processing events in a separate goroutine.
func (p *Processor) Start() {
	p.wg.Add(1)
	go p.processEvents()
}

func (p *Processor) processEvents() {
	defer p.wg.Done()

	for {
		select {
		case event := <-p.eventChan:
			p.handleEvent(event)
		case <-p.done:
			log.Println("Event processor shutting down, processing remaining events...")

			for {
				select {
				case event := <-p.eventChan:
					p.handleEvent(event)
				default:
					log.Println("Finished processing all events")
					return
				}
			}
		}
	}
}

func (p *Processor) handleEvent(event types.OTelEvent) {
	var providerWg sync.WaitGroup
	ctx := context.Background()

	for _, prv := range p.providerRegistry.GetAll() {
		if prv.IsEnabled() {
			providerWg.Add(1)
			go func(p provider.Provider) {
				defer providerWg.Done()
				if err := p.Send(ctx, event); err != nil {
					log.Printf("Error forwarding to %s: %v", p.Name(), err)
				}
			}(prv)
		}
	}
	providerWg.Wait()
}

// EnqueueEvent adds an event to the processing queue. If the queue is full, it drops the event.
func (p *Processor) EnqueueEvent(event types.OTelEvent) {
	select {
	case p.eventChan <- event:
	case <-time.After(100 * time.Millisecond):
		log.Println("Warning: Event processor queue is full, dropping event")
	}
}

// Shutdown gracefully shuts down the event processor, waiting for all events to be processed.
func (p *Processor) Shutdown() error {
	log.Println("Shutting down event processor...")
	close(p.done)

	p.wg.Wait()
	close(p.eventChan)

	log.Println("Event processor shutdown complete")
	return nil
}
