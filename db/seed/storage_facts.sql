INSERT INTO lakehouse_facts (id, master_id, patient_id, resource_type, canonical, codes, timestamp, created_at)
VALUES
  ('33333333-cccc-cccc-cccc-cccccccccccc', 'aaaaaaaa-0000-0000-0000-000000000001', 'patient-001', 'Observation', '{"concept":"blood-glucose","value":115,"unit":"mg/dL","effectiveDateTime":"2025-03-10T09:00:00Z"}', '{"LOINC":"2339-0"}', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
  ('44444444-dddd-dddd-dddd-dddddddddddd', 'bbbbbbbb-0000-0000-0000-000000000002', 'patient-002', 'Observation', '{"concept":"blood-pressure","value":"118/76","unit":"mmHg","effectiveDateTime":"2025-03-11T11:00:00Z"}', '{"LOINC":"85354-9"}', NOW() - INTERVAL '1 days', NOW() - INTERVAL '1 days');

INSERT INTO olap_rollups (id, master_id, patient_id, metric, value, event_time, created_at)
VALUES
  ('55555555-eeee-eeee-eeee-eeeeeeeeeeee', 'aaaaaaaa-0000-0000-0000-000000000001', 'patient-001', 'observation', '{"avg":112,"count":12}', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
  ('66666666-ffff-ffff-ffff-ffffffffffff', 'bbbbbbbb-0000-0000-0000-000000000002', 'patient-002', 'observation', '{"avg":120,"count":15}', NOW() - INTERVAL '1 days', NOW() - INTERVAL '1 days');

INSERT INTO feature_offline_store (id, patient_id, features, version, created_at)
VALUES
  ('patient-001:1', 'patient-001', '{"master_patient_id":"aaaaaaaa-0000-0000-0000-000000000001","concept":"blood-glucose","value":115}', 1, NOW() - INTERVAL '2 days'),
  ('patient-002:1', 'patient-002', '{"master_patient_id":"bbbbbbbb-0000-0000-0000-000000000002","concept":"blood-pressure","value":"118/76"}', 1, NOW() - INTERVAL '1 days');
