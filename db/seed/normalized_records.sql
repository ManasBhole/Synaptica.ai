INSERT INTO normalized_records (id, patient_id, resource_type, canonical, codes, timestamp, created_at)
VALUES
  ('b1d673fa-1111-4e33-aaaa-000000000001', 'patient-001', 'Observation', '{"concept":"blood-glucose","value":110,"unit":"mg/dL","effectiveDateTime":"2025-03-01T09:30:00Z"}', '{"LOINC":"2339-0","SNOMED":"271062007"}', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
  ('b1d673fa-1111-4e33-aaaa-000000000002', 'patient-002', 'Observation', '{"concept":"blood-pressure","value":"120/80","unit":"mmHg","effectiveDateTime":"2025-03-02T09:30:00Z"}', '{"LOINC":"85354-9","SNOMED":"75367002"}', NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
  ('b1d673fa-1111-4e33-aaaa-000000000003', 'patient-001', 'Condition', '{"concept":"diagnosis-diabetes","recordedDate":"2025-02-10","clinicalStatus":"active"}', '{"SNOMED":"44054006","ICD10":"E11.9"}', NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days');
