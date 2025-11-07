INSERT INTO master_patients (id, created_at)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', NOW() - INTERVAL '10 days'),
  ('bbbbbbbb-0000-0000-0000-000000000002', NOW() - INTERVAL '7 days');

INSERT INTO patient_linkages (id, master_id, patient_id, deterministic_key, score, method, attributes, created_at)
VALUES
  ('11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'aaaaaaaa-0000-0000-0000-000000000001', 'patient-001', 'patient_id:patient-001', 1.0, 'deterministic', '{"source":"hospital"}', NOW() - INTERVAL '10 days'),
  ('22222222-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'bbbbbbbb-0000-0000-0000-000000000002', 'patient-002', 'patient_id:patient-002', 1.0, 'deterministic', '{"source":"lab"}', NOW() - INTERVAL '7 days');
