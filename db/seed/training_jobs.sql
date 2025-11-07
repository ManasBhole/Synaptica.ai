INSERT INTO training_jobs (id, model_type, config, filters, status, metrics, artifact_path, created_at, updated_at)
VALUES
  ('77777777-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'risk-score', '{"learning_rate":0.01,"epochs":10}', '{"patient_ids":["patient-001","patient-002"]}', 'completed', '{"auc":0.87,"loss":0.42}', 'artifacts/training/77777777-aaaa-aaaa-aaaa-aaaaaaaaaaaa.json', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
  ('88888888-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'readmission-predictor', '{"learning_rate":0.02,"epochs":12}', '{"cohort":"diabetes"}', 'queued', NULL, NULL, NOW() - INTERVAL '1 days', NOW() - INTERVAL '1 days');
