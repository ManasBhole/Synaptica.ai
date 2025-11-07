INSERT INTO ingestion_requests (id, source, format, payload, status, retry_count, created_at, updated_at)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'hospital', 'fhir', '{"resourceType":"Patient","id":"patient-001"}', 'published', 0, NOW() - INTERVAL '6 days', NOW() - INTERVAL '6 days'),
    ('22222222-2222-2222-2222-222222222222', 'lab', 'hl7', '{"observation":"A1C","value":"6.5"}', 'published', 0, NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
    ('33333333-3333-3333-3333-333333333333', 'imaging', 'dicom', '{"study":"CT","location":"abdomen"}', 'published', 0, NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
    ('44444444-4444-4444-4444-444444444444', 'wearable', 'json', '{"device":"cgm","glucose":110}', 'published', 0, NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
    ('55555555-5555-5555-5555-555555555555', 'telehealth', 'notes', '{"summary":"follow-up scheduled"}', 'published', 0, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
    ('66666666-6666-6666-6666-666666666666', 'hospital', 'fhir', '{"resourceType":"Encounter","id":"enc-204"}', 'published', 0, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
    ('77777777-7777-7777-7777-777777777777', 'lab', 'hl7', '{"observation":"LDL","value":"95"}', 'published', 0, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
    ('88888888-8888-8888-8888-888888888888', 'imaging', 'dicom', '{"study":"MRI","location":"brain"}', 'failed', 1, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    ('99999999-9999-9999-9999-999999999999', 'wearable', 'json', '{"device":"fitness","steps":12345}', 'published', 0, NOW() - INTERVAL '12 hours', NOW() - INTERVAL '12 hours'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'telehealth', 'notes', '{"summary":"remote consult completed"}', 'accepted', 0, NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours');
