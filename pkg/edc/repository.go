package edc

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type studyModel struct {
	ID              uuid.UUID      `gorm:"primaryKey;column:id"`
	Code            string         `gorm:"column:code;uniqueIndex"`
	Name            string         `gorm:"column:name"`
	Phase           string         `gorm:"column:phase"`
	TherapeuticArea string         `gorm:"column:therapeutic_area"`
	Status          string         `gorm:"column:status"`
	Sponsor         string         `gorm:"column:sponsor"`
	ProtocolSummary datatypes.JSON `gorm:"column:protocol_summary"`
	StartDate       *time.Time     `gorm:"column:start_date"`
	EndDate         *time.Time     `gorm:"column:end_date"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
}

func (studyModel) TableName() string { return "studies" }

type siteModel struct {
	ID                    uuid.UUID      `gorm:"primaryKey;column:id"`
	StudyID               uuid.UUID      `gorm:"column:study_id"`
	SiteCode              string         `gorm:"column:site_code"`
	Name                  string         `gorm:"column:name"`
	Country               string         `gorm:"column:country"`
	PrincipalInvestigator string         `gorm:"column:principal_investigator"`
	Status                string         `gorm:"column:status"`
	Contact               datatypes.JSON `gorm:"column:contact"`
	CreatedAt             time.Time      `gorm:"column:created_at"`
}

func (siteModel) TableName() string { return "study_sites" }

type formModel struct {
	ID          uuid.UUID      `gorm:"primaryKey;column:id"`
	StudyID     uuid.UUID      `gorm:"column:study_id"`
	Name        string         `gorm:"column:name"`
	Slug        string         `gorm:"column:slug"`
	Version     int            `gorm:"column:version"`
	Description string         `gorm:"column:description"`
	Schema      datatypes.JSON `gorm:"column:schema"`
	Status      string         `gorm:"column:status"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
}

func (formModel) TableName() string { return "study_forms" }

type visitTemplateModel struct {
	ID          uuid.UUID      `gorm:"primaryKey;column:id"`
	StudyID     uuid.UUID      `gorm:"column:study_id"`
	Name        string         `gorm:"column:name"`
	VisitOrder  int            `gorm:"column:visit_order"`
	WindowStart *int           `gorm:"column:window_start_days"`
	WindowEnd   *int           `gorm:"column:window_end_days"`
	Required    bool           `gorm:"column:required"`
	Forms       datatypes.JSON `gorm:"column:forms"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
}

func (visitTemplateModel) TableName() string { return "visit_templates" }

type subjectModel struct {
	ID               uuid.UUID      `gorm:"primaryKey;column:id"`
	StudyID          uuid.UUID      `gorm:"column:study_id"`
	SiteID           *uuid.UUID     `gorm:"column:site_id"`
	SubjectCode      string         `gorm:"column:subject_code"`
	Status           string         `gorm:"column:status"`
	RandomizationArm string         `gorm:"column:randomization_arm"`
	ConsentedAt      *time.Time     `gorm:"column:consented_at"`
	Demographics     datatypes.JSON `gorm:"column:demographics"`
	CreatedAt        time.Time      `gorm:"column:created_at"`
}

func (subjectModel) TableName() string { return "subjects" }

type visitModel struct {
	ID              uuid.UUID      `gorm:"primaryKey;column:id"`
	SubjectID       uuid.UUID      `gorm:"column:subject_id"`
	VisitTemplateID uuid.UUID      `gorm:"column:visit_template_id"`
	ScheduledDate   *time.Time     `gorm:"column:scheduled_date"`
	ActualDate      *time.Time     `gorm:"column:actual_date"`
	Status          string         `gorm:"column:status"`
	Forms           datatypes.JSON `gorm:"column:forms"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
}

func (visitModel) TableName() string { return "subject_visits" }

type consentVersionModel struct {
	ID           uuid.UUID  `gorm:"primaryKey;column:id"`
	StudyID      uuid.UUID  `gorm:"column:study_id"`
	Version      string     `gorm:"column:version"`
	Title        string     `gorm:"column:title"`
	Summary      string     `gorm:"column:summary"`
	DocumentURL  string     `gorm:"column:document_url"`
	EffectiveAt  time.Time  `gorm:"column:effective_at"`
	SupersededAt *time.Time `gorm:"column:superseded_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

func (consentVersionModel) TableName() string { return "consent_versions" }

type consentSignatureModel struct {
	ID               uuid.UUID      `gorm:"primaryKey;column:id"`
	SubjectID        uuid.UUID      `gorm:"column:subject_id"`
	ConsentVersionID uuid.UUID      `gorm:"column:consent_version_id"`
	SignedAt         time.Time      `gorm:"column:signed_at"`
	SignerName       string         `gorm:"column:signer_name"`
	Method           string         `gorm:"column:method"`
	IPAddress        string         `gorm:"column:ip_address"`
	Metadata         datatypes.JSON `gorm:"column:metadata"`
	CreatedAt        time.Time      `gorm:"column:created_at"`
}

func (consentSignatureModel) TableName() string { return "consent_signatures" }

type auditLogModel struct {
	ID        int64          `gorm:"primaryKey;column:id"`
	StudyID   uuid.UUID      `gorm:"column:study_id"`
	SubjectID *uuid.UUID     `gorm:"column:subject_id"`
	Actor     string         `gorm:"column:actor"`
	Role      string         `gorm:"column:role"`
	Action    string         `gorm:"column:action"`
	Entity    string         `gorm:"column:entity"`
	EntityID  string         `gorm:"column:entity_id"`
	Payload   datatypes.JSON `gorm:"column:payload"`
	CreatedAt time.Time      `gorm:"column:created_at"`
}

func (auditLogModel) TableName() string { return "study_audit_logs" }

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(
		&studyModel{},
		&siteModel{},
		&formModel{},
		&visitTemplateModel{},
		&subjectModel{},
		&visitModel{},
		&consentVersionModel{},
		&consentSignatureModel{},
		&auditLogModel{},
	)
}

func (r *Repository) CreateStudy(ctx context.Context, req models.CreateStudyRequest) (models.Study, error) {
	now := time.Now().UTC()
	study := &studyModel{
		ID:              uuid.New(),
		Code:            req.Code,
		Name:            req.Name,
		Phase:           req.Phase,
		TherapeuticArea: req.TherapeuticArea,
		Status:          "draft",
		Sponsor:         req.Sponsor,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if req.ProtocolSummary != nil {
		if data, err := json.Marshal(req.ProtocolSummary); err == nil {
			study.ProtocolSummary = datatypes.JSON(data)
		}
	}
	if err := r.db.WithContext(ctx).Create(study).Error; err != nil {
		return models.Study{}, err
	}
	return r.buildStudy(ctx, study)
}

func (r *Repository) UpdateStudyStatus(ctx context.Context, studyID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&studyModel{}).Where("id = ?", studyID).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}).Error
}

func (r *Repository) ListStudies(ctx context.Context, limit int) ([]models.Study, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []studyModel
	if err := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	studies := make([]models.Study, 0, len(rows))
	for i := range rows {
		study, err := r.buildStudy(ctx, &rows[i])
		if err != nil {
			return nil, err
		}
		studies = append(studies, study)
	}
	return studies, nil
}

func (r *Repository) GetStudy(ctx context.Context, studyID uuid.UUID) (models.Study, error) {
	var row studyModel
	if err := r.db.WithContext(ctx).First(&row, "id = ?", studyID).Error; err != nil {
		return models.Study{}, err
	}
	return r.buildStudy(ctx, &row)
}

func (r *Repository) buildStudy(ctx context.Context, row *studyModel) (models.Study, error) {
	study := models.Study{
		ID:              row.ID,
		Code:            row.Code,
		Name:            row.Name,
		Phase:           row.Phase,
		TherapeuticArea: row.TherapeuticArea,
		Status:          row.Status,
		Sponsor:         row.Sponsor,
		StartDate:       row.StartDate,
		EndDate:         row.EndDate,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if len(row.ProtocolSummary) > 0 {
		var payload map[string]interface{}
		_ = json.Unmarshal(row.ProtocolSummary, &payload)
		study.ProtocolSummary = payload
	}

	var sites []siteModel
	if err := r.db.WithContext(ctx).Where("study_id = ?", row.ID).Find(&sites).Error; err != nil {
		return models.Study{}, err
	}
	for _, site := range sites {
		study.Sites = append(study.Sites, models.StudySite{
			ID:                    site.ID,
			StudyID:               site.StudyID,
			SiteCode:              site.SiteCode,
			Name:                  site.Name,
			Country:               site.Country,
			PrincipalInvestigator: site.PrincipalInvestigator,
			Status:                site.Status,
			Contact:               jsonMap(site.Contact),
			CreatedAt:             site.CreatedAt,
		})
	}

	var forms []formModel
	if err := r.db.WithContext(ctx).Where("study_id = ?", row.ID).Order("slug, version DESC").Find(&forms).Error; err != nil {
		return models.Study{}, err
	}
	for _, form := range forms {
		study.Forms = append(study.Forms, models.StudyForm{
			ID:          form.ID,
			StudyID:     form.StudyID,
			Name:        form.Name,
			Slug:        form.Slug,
			Version:     form.Version,
			Description: form.Description,
			Schema:      jsonMap(form.Schema),
			Status:      form.Status,
			CreatedAt:   form.CreatedAt,
			UpdatedAt:   form.UpdatedAt,
		})
	}

	var visits []visitTemplateModel
	if err := r.db.WithContext(ctx).Where("study_id = ?", row.ID).Order("visit_order").Find(&visits).Error; err != nil {
		return models.Study{}, err
	}
	for _, vt := range visits {
		study.VisitTemplates = append(study.VisitTemplates, models.VisitTemplate{
			ID:              vt.ID,
			StudyID:         vt.StudyID,
			Name:            vt.Name,
			VisitOrder:      vt.VisitOrder,
			WindowStartDays: vt.WindowStart,
			WindowEndDays:   vt.WindowEnd,
			Required:        vt.Required,
			Forms:           jsonStringArray(vt.Forms),
			CreatedAt:       vt.CreatedAt,
			UpdatedAt:       vt.UpdatedAt,
		})
	}

	var counts struct {
		Active int
		Total  int
	}
	query := `SELECT SUM(CASE WHEN status IN ('enrolled','active','randomized') THEN 1 ELSE 0 END) AS active, COUNT(*) AS total FROM subjects WHERE study_id = ?`
	r.db.WithContext(ctx).Raw(query, row.ID).Scan(&counts)
	study.ActiveSubjects = counts.Active
	study.TotalSubjects = counts.Total

	return study, nil
}

func (r *Repository) CreateSite(ctx context.Context, studyID uuid.UUID, req models.CreateStudySiteRequest) (models.StudySite, error) {
	site := &siteModel{
		ID:                    uuid.New(),
		StudyID:               studyID,
		SiteCode:              req.SiteCode,
		Name:                  req.Name,
		Country:               req.Country,
		PrincipalInvestigator: req.PrincipalInvestigator,
		Status:                "planned",
		CreatedAt:             time.Now().UTC(),
	}
	if req.Contact != nil {
		if data, err := json.Marshal(req.Contact); err == nil {
			site.Contact = datatypes.JSON(data)
		}
	}
	if err := r.db.WithContext(ctx).Create(site).Error; err != nil {
		return models.StudySite{}, err
	}
	return models.StudySite{
		ID:                    site.ID,
		StudyID:               site.StudyID,
		SiteCode:              site.SiteCode,
		Name:                  site.Name,
		Country:               site.Country,
		PrincipalInvestigator: site.PrincipalInvestigator,
		Status:                site.Status,
		Contact:               jsonMap(site.Contact),
		CreatedAt:             site.CreatedAt,
	}, nil
}

func (r *Repository) CreateForm(ctx context.Context, studyID uuid.UUID, req models.CreateStudyFormRequest) (models.StudyForm, error) {
	version := 1
	var latest formModel
	if err := r.db.WithContext(ctx).Where("study_id = ? AND slug = ?", studyID, req.Slug).Order("version DESC").First(&latest).Error; err == nil {
		version = latest.Version + 1
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.StudyForm{}, err
	}

	form := &formModel{
		ID:          uuid.New(),
		StudyID:     studyID,
		Name:        req.Name,
		Slug:        req.Slug,
		Version:     version,
		Description: req.Description,
		Status:      defaultString(req.Status, "draft"),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if req.Schema != nil {
		if data, err := json.Marshal(req.Schema); err == nil {
			form.Schema = datatypes.JSON(data)
		}
	}

	if err := r.db.WithContext(ctx).Create(form).Error; err != nil {
		return models.StudyForm{}, err
	}

	return models.StudyForm{
		ID:          form.ID,
		StudyID:     form.StudyID,
		Name:        form.Name,
		Slug:        form.Slug,
		Version:     form.Version,
		Description: form.Description,
		Schema:      req.Schema,
		Status:      form.Status,
		CreatedAt:   form.CreatedAt,
		UpdatedAt:   form.UpdatedAt,
	}, nil
}

func (r *Repository) CreateVisitTemplate(ctx context.Context, studyID uuid.UUID, req models.CreateVisitTemplateRequest) (models.VisitTemplate, error) {
	required := true
	if req.Required != nil {
		required = *req.Required
	}
	vt := &visitTemplateModel{
		ID:          uuid.New(),
		StudyID:     studyID,
		Name:        req.Name,
		VisitOrder:  req.VisitOrder,
		WindowStart: req.WindowStartDays,
		WindowEnd:   req.WindowEndDays,
		Required:    required,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if req.Forms != nil {
		if data, err := json.Marshal(req.Forms); err == nil {
			vt.Forms = datatypes.JSON(data)
		}
	}
	if err := r.db.WithContext(ctx).Create(vt).Error; err != nil {
		return models.VisitTemplate{}, err
	}
	return models.VisitTemplate{
		ID:              vt.ID,
		StudyID:         vt.StudyID,
		Name:            vt.Name,
		VisitOrder:      vt.VisitOrder,
		WindowStartDays: vt.WindowStart,
		WindowEndDays:   vt.WindowEnd,
		Required:        vt.Required,
		Forms:           req.Forms,
		CreatedAt:       vt.CreatedAt,
		UpdatedAt:       vt.UpdatedAt,
	}, nil
}

func (r *Repository) CreateConsentVersion(ctx context.Context, studyID uuid.UUID, req models.CreateConsentVersionRequest) (models.ConsentVersion, error) {
	entry := &consentVersionModel{
		ID:           uuid.New(),
		StudyID:      studyID,
		Version:      req.Version,
		Title:        req.Title,
		Summary:      req.Summary,
		DocumentURL:  req.DocumentURL,
		EffectiveAt:  req.EffectiveAt,
		SupersededAt: req.SupersededAt,
		CreatedAt:    time.Now().UTC(),
	}
	if err := r.db.WithContext(ctx).Create(entry).Error; err != nil {
		return models.ConsentVersion{}, err
	}
	return models.ConsentVersion{
		ID:           entry.ID,
		StudyID:      entry.StudyID,
		Version:      entry.Version,
		Title:        entry.Title,
		Summary:      entry.Summary,
		DocumentURL:  entry.DocumentURL,
		EffectiveAt:  entry.EffectiveAt,
		SupersededAt: entry.SupersededAt,
		CreatedAt:    entry.CreatedAt,
	}, nil
}

func (r *Repository) ListConsentVersions(ctx context.Context, studyID uuid.UUID, limit int) ([]models.ConsentVersion, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []consentVersionModel
	if err := r.db.WithContext(ctx).Where("study_id = ?", studyID).Order("effective_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	versions := make([]models.ConsentVersion, 0, len(rows))
	for _, row := range rows {
		versions = append(versions, models.ConsentVersion{
			ID:           row.ID,
			StudyID:      row.StudyID,
			Version:      row.Version,
			Title:        row.Title,
			Summary:      row.Summary,
			DocumentURL:  row.DocumentURL,
			EffectiveAt:  row.EffectiveAt,
			SupersededAt: row.SupersededAt,
			CreatedAt:    row.CreatedAt,
		})
	}
	return versions, nil
}

func (r *Repository) EnrollSubject(ctx context.Context, studyID uuid.UUID, req models.EnrollSubjectRequest) (models.Subject, error) {
	subject := &subjectModel{
		ID:               uuid.New(),
		StudyID:          studyID,
		SiteID:           req.SiteID,
		SubjectCode:      req.SubjectCode,
		Status:           "screening",
		RandomizationArm: req.RandomizationArm,
		CreatedAt:        time.Now().UTC(),
	}
	if req.Demographics != nil {
		if data, err := json.Marshal(req.Demographics); err == nil {
			subject.Demographics = datatypes.JSON(data)
		}
	}

	if err := r.db.WithContext(ctx).Create(subject).Error; err != nil {
		return models.Subject{}, err
	}

	return models.Subject{
		ID:               subject.ID,
		StudyID:          subject.StudyID,
		SiteID:           subject.SiteID,
		SubjectCode:      subject.SubjectCode,
		Status:           subject.Status,
		RandomizationArm: subject.RandomizationArm,
		Demographics:     req.Demographics,
		CreatedAt:        subject.CreatedAt,
	}, nil
}

func (r *Repository) GetSubject(ctx context.Context, subjectID uuid.UUID) (models.Subject, error) {
	var row subjectModel
	if err := r.db.WithContext(ctx).First(&row, "id = ?", subjectID).Error; err != nil {
		return models.Subject{}, err
	}
	return models.Subject{
		ID:               row.ID,
		StudyID:          row.StudyID,
		SiteID:           row.SiteID,
		SubjectCode:      row.SubjectCode,
		Status:           row.Status,
		RandomizationArm: row.RandomizationArm,
		ConsentedAt:      row.ConsentedAt,
		Demographics:     jsonMap(row.Demographics),
		CreatedAt:        row.CreatedAt,
	}, nil
}

func (r *Repository) RecordConsent(ctx context.Context, subjectID uuid.UUID, req models.ConsentSignatureRequest) (models.ConsentSignature, error) {
	var subject subjectModel
	if err := r.db.WithContext(ctx).First(&subject, "id = ?", subjectID).Error; err != nil {
		return models.ConsentSignature{}, err
	}

	signature := &consentSignatureModel{
		ID:               uuid.New(),
		SubjectID:        subjectID,
		ConsentVersionID: req.ConsentVersionID,
		SignedAt:         req.SignedAt,
		SignerName:       req.SignerName,
		Method:           req.Method,
		IPAddress:        req.IPAddress,
		CreatedAt:        time.Now().UTC(),
	}
	if req.Metadata != nil {
		if data, err := json.Marshal(req.Metadata); err == nil {
			signature.Metadata = datatypes.JSON(data)
		}
	}
	if err := r.db.WithContext(ctx).Create(signature).Error; err != nil {
		return models.ConsentSignature{}, err
	}

	_ = r.db.WithContext(ctx).Model(&subjectModel{}).Where("id = ?", subjectID).Updates(map[string]interface{}{
		"consented_at": req.SignedAt,
		"status":       "enrolled",
	})

	return models.ConsentSignature{
		ID:               signature.ID,
		SubjectID:        signature.SubjectID,
		ConsentVersionID: signature.ConsentVersionID,
		SignedAt:         signature.SignedAt,
		SignerName:       signature.SignerName,
		Method:           signature.Method,
		IPAddress:        signature.IPAddress,
		Metadata:         req.Metadata,
		CreatedAt:        signature.CreatedAt,
	}, nil
}

func (r *Repository) ListSubjects(ctx context.Context, studyID uuid.UUID, limit int) ([]models.Subject, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []subjectModel
	if err := r.db.WithContext(ctx).Where("study_id = ?", studyID).Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	subjects := make([]models.Subject, 0, len(rows))
	for _, row := range rows {
		subjects = append(subjects, models.Subject{
			ID:               row.ID,
			StudyID:          row.StudyID,
			SiteID:           row.SiteID,
			SubjectCode:      row.SubjectCode,
			Status:           row.Status,
			RandomizationArm: row.RandomizationArm,
			ConsentedAt:      row.ConsentedAt,
			Demographics:     jsonMap(row.Demographics),
			CreatedAt:        row.CreatedAt,
		})
	}
	return subjects, nil
}

func (r *Repository) ListAuditLogs(ctx context.Context, studyID uuid.UUID, limit int) ([]models.AuditLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []auditLogModel
	if err := r.db.WithContext(ctx).Where("study_id = ?", studyID).Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	logs := make([]models.AuditLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, models.AuditLog{
			ID:        row.ID,
			StudyID:   row.StudyID,
			SubjectID: row.SubjectID,
			Actor:     row.Actor,
			Role:      row.Role,
			Action:    row.Action,
			Entity:    row.Entity,
			EntityID:  row.EntityID,
			Payload:   jsonMap(row.Payload),
			CreatedAt: row.CreatedAt,
		})
	}
	return logs, nil
}

func (r *Repository) AppendAuditLog(ctx context.Context, log models.AuditLog) error {
	payload, _ := json.Marshal(log.Payload)
	entry := &auditLogModel{
		StudyID:   log.StudyID,
		SubjectID: log.SubjectID,
		Actor:     log.Actor,
		Role:      log.Role,
		Action:    log.Action,
		Entity:    log.Entity,
		EntityID:  log.EntityID,
		Payload:   datatypes.JSON(payload),
		CreatedAt: time.Now().UTC(),
	}
	return r.db.WithContext(ctx).Create(entry).Error
}

func jsonMap(data datatypes.JSON) map[string]interface{} {
	if len(data) == 0 {
		return nil
	}
	var result map[string]interface{}
	_ = json.Unmarshal(data, &result)
	return result
}

func jsonStringArray(data datatypes.JSON) []string {
	if len(data) == 0 {
		return nil
	}
	var result []string
	_ = json.Unmarshal(data, &result)
	return result
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
