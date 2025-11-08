INSERT INTO cohort_templates (id, tenant_id, name, description, dsl, tags, created_at)
VALUES
  ('aaaaaaaa-1111-2222-3333-444444444444', NULL, 'Hypertension High Risk', 'Patients with systolic BP above 140 in the last 30 days',
   'select patient_id, resource_type, concept, value, timestamp
from lakehouse
where resource_type = ''observation'' and concept = ''blood_pressure_systolic'' and value > 140 and timestamp >= NOW() - INTERVAL ''30 days''
limit 500',
   ARRAY['risk','blood-pressure'], NOW()),
  ('bbbbbbbb-2222-3333-4444-555555555555', NULL, 'Recent Diabetes Labs', 'Patients with HbA1c recorded in current quarter',
   'select patient_id, concept, value, timestamp
from lakehouse
where concept = ''hba1c'' and timestamp >= date_trunc(''quarter'', NOW())
limit 300',
   ARRAY['lab','diabetes'], NOW()),
  ('cccccccc-3333-4444-5555-666666666666', NULL, 'Cardiology Encounters', 'Cardio visits in the last 90 days',
   'select patient_id, resource_type, timestamp
from lakehouse
where resource_type = ''encounter'' and concept = ''cardiology'' and timestamp >= NOW() - INTERVAL ''90 days''
limit 400',
   ARRAY['encounter','cardio'], NOW());
