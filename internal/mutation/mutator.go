package mutation

import (
	"sort"
	"sync"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/settings"
	"VRCFaceTracking/pkg/tracking"
)

const enabledKey = "MutatorEnabled"

// Mutator owns the ordered list of mutations and applies them to a private
// copy of the incoming data.
type Mutator struct {
	logger *zap.Logger
	store  *settings.Store

	mu        sync.Mutex
	enabled   bool
	mutations []Mutation
	buffer    *tracking.TrackingData
}

// NewMutator creates a mutator with the built-in mutations, sorted by
// priority.
func NewMutator(logger *zap.Logger, store *settings.Store) *Mutator {
	m := &Mutator{
		logger: logger.Named("Mutator"),
		store:  store,
		buffer: tracking.NewTrackingData(),
		mutations: []Mutation{
			NewParameterAdjustment(),
			NewCorrectors(),
			NewFilter(),
		},
	}
	sort.SliceStable(m.mutations, func(i, j int) bool {
		return m.mutations[i].Priority() < m.mutations[j].Priority()
	})
	return m
}

func (m *Mutator) Enabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.enabled
}

func (m *Mutator) SetEnabled(enabled bool) {
	m.mu.Lock()
	m.enabled = enabled
	m.mu.Unlock()
	_ = m.store.Set(enabledKey, enabled)
}

// Mutations returns the ordered mutation list.
func (m *Mutator) Mutations() []Mutation {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Mutation(nil), m.mutations...)
}

// Find returns the mutation with the given name.
func (m *Mutator) Find(name string) (Mutation, bool) {
	for _, mut := range m.Mutations() {
		if mut.Name() == name {
			return mut, true
		}
	}
	return nil, false
}

// Mutate returns the mutated data. When disabled the input is returned
// unchanged.
func (m *Mutator) Mutate(input *tracking.TrackingData) *tracking.TrackingData {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.enabled {
		return input
	}
	m.buffer.CopyFrom(input)
	for _, mut := range m.mutations {
		if mut.Active() {
			mut.Mutate(m.buffer)
		}
	}
	return m.buffer
}

// Load restores the enabled flag and every persisted mutation, then
// initialises all mutations against data.
func (m *Mutator) Load(data *tracking.TrackingData) {
	m.logger.Debug("Loading mutation data...")
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = settings.GetOr(m.store, enabledKey, true)
	for _, mut := range m.mutations {
		if !mut.Persisted() {
			continue
		}
		found, err := m.store.Get(mut.Name(), mut)
		if err != nil {
			m.logger.Error("Creating new mutation data", zap.String("mutation", mut.Name()), zap.Error(err))
		} else if found {
			m.logger.Info("Loaded mutation", zap.String("mutation", mut.Name()))
		}
	}
	for _, mut := range m.mutations {
		m.logger.Info("Initializing mutation", zap.String("mutation", mut.Name()))
		mut.Initialize(data)
	}
	m.logger.Debug("Mutations initialized successfully.")
}

// Save persists the enabled flag and every persisted mutation.
func (m *Mutator) Save() error {
	m.logger.Debug("Saving mutation configuration...")
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.store.Set(enabledKey, m.enabled); err != nil {
		return err
	}
	for _, mut := range m.mutations {
		if err := m.saveLocked(mut); err != nil {
			return err
		}
	}
	return nil
}

// SaveMutation persists a single mutation.
func (m *Mutator) SaveMutation(mut Mutation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked(mut)
}

func (m *Mutator) saveLocked(mut Mutation) error {
	if !mut.Persisted() {
		return nil
	}
	return m.store.Set(mut.Name(), mut)
}
