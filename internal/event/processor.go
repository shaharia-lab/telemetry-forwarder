package event

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/shaharia-lab/telemetry-forwarder/internal/provider"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
)

type EventProcessor struct {
	providerRegistry *provider.Registry
	eventChan        chan types.OTelEvent
	done             chan struct{}
	wg               sync.WaitGroup
}

func NewEventProcessor(providerRegistry *provider.Registry, bufferSize int) *EventProcessor {
	return &EventProcessor{
		providerRegistry: providerRegistry,
		eventChan:        make(chan types.OTelEvent, bufferSize),
		done:             make(chan struct{}),
	}
}

func (p *EventProcessor) Start() {
	p.wg.Add(1)
	go p.processEvents()
}

func (p *EventProcessor) processEvents() {
	defer p.wg.Done()

	for {
		select {
		case event := <-p.eventChan:
			p.handleEvent(event)
		case <-p.done:
			// Process any remaining events in the channel
			log.Println("Event processor shutting down, processing remaining events...")

			// Process any remaining events
			for {
				select {
				case event := <-p.eventChan:
					p.handleEvent(event)
				default:
					// No more events in the channel
					log.Println("Finished processing all events")
					return
				}
			}
		}
	}
}

func (p *EventProcessor) handleEvent(event types.OTelEvent) {
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

func (p *EventProcessor) EnqueueEvent(event types.OTelEvent) {
	// Non-blocking send with timeout in case the channel is full
	select {
	case p.eventChan <- event:
		// Event successfully enqueued
	case <-time.After(100 * time.Millisecond):
		log.Println("Warning: Event processor queue is full, dropping event")
	}
}

func (p *EventProcessor) Shutdown() error {
	log.Println("Shutting down event processor...")
	close(p.done)

	// Wait for the processor goroutine to finish
	p.wg.Wait()
	close(p.eventChan)

	log.Println("Event processor shutdown complete")
	return nil
}
