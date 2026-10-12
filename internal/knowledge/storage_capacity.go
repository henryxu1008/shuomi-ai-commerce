package knowledge

import (
	"context"
	"task-processor/internal/ledger/storagecapacity"
)

func (s *Service) ConfigureStorageCapacity(m *storagecapacity.Meter)   { s.capacity = m }
func (p *Processor) ConfigureStorageCapacity(m *storagecapacity.Meter) { p.capacity = m }

func (s *Service) StorageCapacityConfigured() bool {
	return s != nil && s.capacity != nil && s.capacity.Writer != nil && storagecapacity.ValidID(s.capacity.Namespace, 256)
}
func capacityObject(r Revision) storagecapacity.Object {
	return storagecapacity.Object{Key: r.ObjectKey, SHA256: r.SHA256, SizeBytes: r.SizeBytes, Category: "knowledge_document"}
}
func confirmCapacity(ctx context.Context, m *storagecapacity.Meter, r Revision) error {
	if m == nil {
		return nil
	}
	o := capacityObject(r)
	o.Namespace = m.Namespace
	reservation, e := m.Writer.FindObjectWrite(ctx, r.OrganizationID, "knowledge", o)
	if e != nil {
		return e
	}
	return m.Confirm(ctx, r.OrganizationID, reservation, capacityObject(r))
}
