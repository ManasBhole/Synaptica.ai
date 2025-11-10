package edc

import (
	"context"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateStudy(ctx context.Context, req models.CreateStudyRequest, actor string) (models.Study, error) {
	study, err := s.repo.CreateStudy(ctx, req)
	if err != nil {
		return models.Study{}, err
	}
	_ = s.log(ctx, models.AuditLog{
		StudyID:  study.ID,
		Actor:    actor,
		Action:   "study_created",
		Entity:   "study",
		EntityID: study.ID.String(),
		Payload:  map[string]interface{}{"code": study.Code, "name": study.Name},
	})
	return study, nil
}

func (s *Service) UpdateStudyStatus(ctx context.Context, studyID uuid.UUID, status string, actor string) error {
	if err := s.repo.UpdateStudyStatus(ctx, studyID, status); err != nil {
		return err
	}
	return s.log(ctx, models.AuditLog{
		StudyID:  studyID,
		Actor:    actor,
		Action:   "study_status_updated",
		Entity:   "study",
		EntityID: studyID.String(),
		Payload:  map[string]interface{}{"status": status},
	})
}

func (s *Service) ListStudies(ctx context.Context, limit int) ([]models.Study, error) {
	return s.repo.ListStudies(ctx, limit)
}

func (s *Service) GetStudy(ctx context.Context, studyID uuid.UUID) (models.Study, error) {
	return s.repo.GetStudy(ctx, studyID)
}

func (s *Service) CreateSite(ctx context.Context, studyID uuid.UUID, req models.CreateStudySiteRequest, actor string) (models.StudySite, error) {
	site, err := s.repo.CreateSite(ctx, studyID, req)
	if err != nil {
		return models.StudySite{}, err
	}
	_ = s.log(ctx, models.AuditLog{
		StudyID:  studyID,
		Actor:    actor,
		Action:   "site_created",
		Entity:   "site",
		EntityID: site.ID.String(),
		Payload:  map[string]interface{}{"site_code": site.SiteCode, "name": site.Name},
	})
	return site, nil
}

func (s *Service) CreateForm(ctx context.Context, studyID uuid.UUID, req models.CreateStudyFormRequest, actor string) (models.StudyForm, error) {
	form, err := s.repo.CreateForm(ctx, studyID, req)
	if err != nil {
		return models.StudyForm{}, err
	}
	_ = s.log(ctx, models.AuditLog{
		StudyID:  studyID,
		Actor:    actor,
		Action:   "form_created",
		Entity:   "form",
		EntityID: form.ID.String(),
		Payload:  map[string]interface{}{"slug": form.Slug, "version": form.Version},
	})
	return form, nil
}

func (s *Service) CreateVisitTemplate(ctx context.Context, studyID uuid.UUID, req models.CreateVisitTemplateRequest, actor string) (models.VisitTemplate, error) {
	visit, err := s.repo.CreateVisitTemplate(ctx, studyID, req)
	if err != nil {
		return models.VisitTemplate{}, err
	}
	_ = s.log(ctx, models.AuditLog{
		StudyID:  studyID,
		Actor:    actor,
		Action:   "visit_template_created",
		Entity:   "visit_template",
		EntityID: visit.ID.String(),
		Payload:  map[string]interface{}{"order": visit.VisitOrder, "name": visit.Name},
	})
	return visit, nil
}

func (s *Service) CreateConsentVersion(ctx context.Context, studyID uuid.UUID, req models.CreateConsentVersionRequest, actor string) (models.ConsentVersion, error) {
	version, err := s.repo.CreateConsentVersion(ctx, studyID, req)
	if err != nil {
		return models.ConsentVersion{}, err
	}
	_ = s.log(ctx, models.AuditLog{
		StudyID:  studyID,
		Actor:    actor,
		Action:   "consent_version_created",
		Entity:   "consent_version",
		EntityID: version.ID.String(),
		Payload: map[string]interface{}{
			"version":      version.Version,
			"effective_at": version.EffectiveAt,
		},
	})
	return version, nil
}

func (s *Service) ListConsentVersions(ctx context.Context, studyID uuid.UUID, limit int) ([]models.ConsentVersion, error) {
	return s.repo.ListConsentVersions(ctx, studyID, limit)
}

func (s *Service) EnrollSubject(ctx context.Context, studyID uuid.UUID, req models.EnrollSubjectRequest, actor string) (models.Subject, error) {
	subject, err := s.repo.EnrollSubject(ctx, studyID, req)
	if err != nil {
		return models.Subject{}, err
	}
	_ = s.log(ctx, models.AuditLog{
		StudyID:   studyID,
		SubjectID: &subject.ID,
		Actor:     actor,
		Action:    "subject_enrolled",
		Entity:    "subject",
		EntityID:  subject.ID.String(),
		Payload:   map[string]interface{}{"subject_code": subject.SubjectCode},
	})
	return subject, nil
}

func (s *Service) RecordConsent(ctx context.Context, subjectID uuid.UUID, req models.ConsentSignatureRequest, actor string) (models.ConsentSignature, error) {
	subject, err := s.repo.GetSubject(ctx, subjectID)
	if err != nil {
		return models.ConsentSignature{}, err
	}
	signature, err := s.repo.RecordConsent(ctx, subjectID, req)
	if err != nil {
		return models.ConsentSignature{}, err
	}
	_ = s.log(ctx, models.AuditLog{
		StudyID:   subject.StudyID,
		SubjectID: &signature.SubjectID,
		Actor:     actor,
		Action:    "consent_signed",
		Entity:    "consent",
		EntityID:  signature.ID.String(),
		Payload:   map[string]interface{}{"consent_version_id": signature.ConsentVersionID},
	})
	return signature, nil
}

func (s *Service) ListSubjects(ctx context.Context, studyID uuid.UUID, limit int) ([]models.Subject, error) {
	return s.repo.ListSubjects(ctx, studyID, limit)
}

func (s *Service) ListAuditLogs(ctx context.Context, studyID uuid.UUID, limit int) ([]models.AuditLog, error) {
	return s.repo.ListAuditLogs(ctx, studyID, limit)
}

func (s *Service) log(ctx context.Context, entry models.AuditLog) error {
	if entry.Actor == "" {
		entry.Actor = "system"
	}
	if entry.Payload == nil {
		entry.Payload = map[string]interface{}{}
	}
	if entry.StudyID == uuid.Nil {
		logger.Log.Warn("audit log missing study id")
		return nil
	}
	return s.repo.AppendAuditLog(ctx, entry)
}
