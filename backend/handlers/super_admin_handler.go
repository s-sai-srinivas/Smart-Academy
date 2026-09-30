package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"coding-platform/services"
	"coding-platform/utils"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ─── Request / Response types ────────────────────────────────────────

type CreateCollegeRequest struct {
	CollegeID   string `json:"college_id" binding:"required,min=2,max=50"`
	CollegeName string `json:"college_name" binding:"required,min=2,max=200"`
	ShortName   string `json:"short_name" binding:"required,min=2,max=50"`
	Address     string `json:"address"` // Optional, max size checked in middleware
	// College admin credentials (created atomically)
	AdminRegdNo   string `json:"admin_regdno" binding:"required,min=3,max=50"`
	AdminEmail    string `json:"admin_email" binding:"required,email,max=255"`
	AdminPassword string `json:"admin_password" binding:"required,min=6,max=128"`
	AdminName     string `json:"admin_name" binding:"required,min=1,max=200"`
}

type UpdateCollegeStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// ─── Handlers ────────────────────────────────────────────────────────

// CreateCollege creates a new college and its college_admin in a single transaction.
// Only super_admin can call this (enforced by middleware).
func CreateCollege(c *gin.Context) {
	var req CreateCollegeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate admin password strength
	if err := utils.ValidatePassword(req.AdminPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check for duplicate college_id
	var existingByID models.College
	if err := database.DB.Where("college_id = ?", req.CollegeID).First(&existingByID).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("College with ID '%s' already exists", req.CollegeID)})
		return
	}

	// Check for duplicate short_name
	var existing models.College
	if err := database.DB.Where("short_name = ?", req.ShortName).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("College with short name '%s' already exists", req.ShortName)})
		return
	}

	// Check admin regdno / email uniqueness
	var existingUser models.User
	if err := database.DB.Where("regdno = ?", req.AdminRegdNo).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Admin registration number already exists"})
		return
	}
	if err := database.DB.Where("email = ?", req.AdminEmail).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Admin email already exists"})
		return
	}

	// Use a transaction to create college + admin atomically
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Create college
	college := models.College{
		CollegeID:   req.CollegeID,
		CollegeName: req.CollegeName,
		ShortName:   req.ShortName,
		Address:     req.Address,
		IsActive:    true,
	}
	if err := tx.Create(&college).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create college"})
		return
	}

	// Create college_admin user linked to the new college
	adminUser := models.User{
		RegdNo:    req.AdminRegdNo,
		Email:     req.AdminEmail,
		Name:      req.AdminName,
		Role:      "college_admin",
		CollegeID: &college.CollegeID,
		IsActive:  true,
	}
	if err := adminUser.HashPassword(req.AdminPassword); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash admin password"})
		return
	}
	if err := tx.Create(&adminUser).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create college admin"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "College and admin created successfully",
		"college": college,
		"admin": gin.H{
			"regdno": adminUser.RegdNo,
			"email":  adminUser.Email,
			"name":   adminUser.Name,
			"role":   adminUser.Role,
		},
	})
}

// ListAllColleges returns every college (active and inactive) for super_admin.
func ListAllColleges(c *gin.Context) {
	var colleges []models.College
	query := database.DB.Order("college_name")

	// Optional filter: ?active=true / ?active=false
	if activeParam := c.Query("active"); activeParam != "" {
		isActive := activeParam == "true"
		query = query.Where("is_active = ?", isActive)
	}

	if err := query.Find(&colleges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch colleges"})
		return
	}

	// Attach admin count for each college
	type collegeResponse struct {
		models.College
		AdminCount int64 `json:"admin_count"`
		UserCount  int64 `json:"user_count"`
	}

	results := make([]collegeResponse, 0, len(colleges))
	for _, col := range colleges {
		var adminCount, userCount int64
		database.DB.Model(&models.User{}).
			Where("college_id = ? AND role = ? AND is_active = true", col.CollegeID, "college_admin").
			Count(&adminCount)
		database.DB.Model(&models.User{}).
			Where("college_id = ? AND is_active = true", col.CollegeID).
			Count(&userCount)
		results = append(results, collegeResponse{
			College:    col,
			AdminCount: adminCount,
			UserCount:  userCount,
		})
	}

	c.JSON(http.StatusOK, results)
}

// GetCollegeByID returns a single college with summary stats.
func GetCollegeByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid college ID"})
		return
	}

	var college models.College
	if err := database.DB.First(&college, "college_id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "College not found"})
		return
	}

	var adminCount, studentCount, facultyCount int64
	database.DB.Model(&models.User{}).Where("college_id = ? AND role = ? AND is_active = true", id, "college_admin").Count(&adminCount)
	database.DB.Model(&models.User{}).Where("college_id = ? AND role = ? AND is_active = true", id, "student").Count(&studentCount)
	database.DB.Model(&models.User{}).Where("college_id = ? AND role IN (?, ?) AND is_active = true", id, "faculty", "hod").Count(&facultyCount)

	c.JSON(http.StatusOK, gin.H{
		"college":       college,
		"admin_count":   adminCount,
		"student_count": studentCount,
		"faculty_count": facultyCount,
	})
}

// UpdateCollegeStatus enables or disables (suspends) a college.
func UpdateCollegeStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid college ID"})
		return
	}

	var req UpdateCollegeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var college models.College
	if err := database.DB.First(&college, "college_id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "College not found"})
		return
	}

	// Check if status is actually changing
	statusChanged := college.IsActive != req.IsActive
	wasActive := college.IsActive

	college.IsActive = req.IsActive
	if err := database.DB.Save(&college).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update college status"})
		return
	}

	// If college was suspended (active -> inactive), force logout all users
	if statusChanged && wasActive && !req.IsActive {
		hub := services.GetWebSocketHub()
		hub.ForceLogoutCollege(id, "College suspended")
	}

	status := "activated"
	if !req.IsActive {
		status = "suspended"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("College %s successfully", status),
		"college": college,
	})
}

// =====================================================
// Global Practice Problems Management
// =====================================================

// CreateGlobalProblemRequest represents the request for creating a global problem
type CreateGlobalProblemRequest struct {
	SubjectID   *uint  `json:"subject_id"`
	Title       string `json:"title" binding:"required,min=3,max=300"`
	Description string `json:"description" binding:"required"`
	Difficulty  string `json:"difficulty" binding:"required,oneof=easy medium hard"`
	Tags        string `json:"tags"`
	TimeLimit   int    `json:"time_limit"`
	MemoryLimit int    `json:"memory_limit"`
	Points      int    `json:"points"`
}

// CreateGlobalProblem creates a new global problem (super admin only)
// Global problems are accessible to all students across all colleges
func CreateGlobalProblem(c *gin.Context) {
	var req CreateGlobalProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate description size
	const maxDescriptionSize = 50 * 1024 // 50KB
	if len(req.Description) > maxDescriptionSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Description exceeds maximum size of 50KB"})
		return
	}

	userRegdNo, _ := c.Get("regdno")

	// Create problem with IsGlobal = true and no college_id
	problem := models.Problem{
		Title:       req.Title,
		Description: req.Description,
		Difficulty:  req.Difficulty,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		Tags:        req.Tags,
		Points:      req.Points,
		IsGlobal:    true,
		CollegeID:   nil, // Global problems have no college
		SubjectID:   req.SubjectID,
		CreatedBy:   userRegdNo.(string),
	}

	// Set defaults
	if problem.TimeLimit == 0 {
		problem.TimeLimit = 2000
	}
	if problem.MemoryLimit == 0 {
		problem.MemoryLimit = 256000
	}

	if err := database.DB.Create(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create global problem"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Global problem created successfully",
		"problem": problem,
	})
}

// UpdateGlobalProblem updates an existing global problem
func UpdateGlobalProblem(c *gin.Context) {
	id := c.Param("id")

	var problem models.Problem
	if err := database.DB.Where("id = ? AND is_global = ?", id, true).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Global problem not found"})
		return
	}

	var req CreateGlobalProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	problem.Title = req.Title
	problem.Description = req.Description
	problem.Difficulty = req.Difficulty
	problem.TimeLimit = req.TimeLimit
	problem.MemoryLimit = req.MemoryLimit
	problem.Tags = req.Tags
	problem.Points = req.Points
	problem.SubjectID = req.SubjectID

	if err := database.DB.Save(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update global problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"problem": problem})
}

// DeleteGlobalProblem deletes a global problem
func DeleteGlobalProblem(c *gin.Context) {
	id := c.Param("id")

	var problem models.Problem
	if err := database.DB.Where("id = ? AND is_global = ?", id, true).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Global problem not found"})
		return
	}

	// Delete associated test cases first
	database.DB.Where("problem_id = ?", problem.ID).Delete(&models.TestCase{})

	// Delete the problem
	if err := database.DB.Delete(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete global problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Global problem deleted successfully"})
}

// GetGlobalProblems returns all global problems for super admin management
func GetGlobalProblems(c *gin.Context) {
	var problems []models.Problem
	if err := database.DB.Where("is_global = ?", true).
		Preload("TestCases").
		Order("created_at DESC").
		Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch global problems"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"problems": problems})
}

// CreateGlobalProblemTestCase creates a test case for a global problem
func CreateGlobalProblemTestCase(c *gin.Context) {
	problemID := c.Param("id")

	// Verify the problem exists and is global
	var problem models.Problem
	if err := database.DB.Where("id = ? AND is_global = ?", problemID, true).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Global problem not found"})
		return
	}

	var req struct {
		Input          string `json:"input" binding:"required"`
		ExpectedOutput string `json:"expected_output" binding:"required"`
		IsSample       bool   `json:"is_sample"`
		Points         int    `json:"points"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pid, _ := strconv.Atoi(problemID)
	testCase := models.TestCase{
		ProblemID:      uint(pid),
		Input:          req.Input,
		ExpectedOutput: req.ExpectedOutput,
		IsSample:       req.IsSample,
		Points:         req.Points,
	}

	if testCase.Points == 0 {
		testCase.Points = 10
	}

	if err := database.DB.Create(&testCase).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create test case"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"test_case": testCase})
}

// =====================================================
// Subject & Topic Management (Super Admin only)
// =====================================================

// ─── Subject CRUD ────────────────────────────────────────────────────

// CreateSubjectRequest represents the request for creating a subject
type CreateSubjectRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description"`
}

// CreateSubject creates a new subject (super admin only)
func CreateSubject(c *gin.Context) {
	var req CreateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject name cannot be empty"})
		return
	}

	var existing models.Subject
	if err := database.DB.Where("LOWER(name) = LOWER(?)", name).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Subject '%s' already exists", name)})
		return
	}

	userRegdNo, _ := c.Get("regdno")

	subject := models.Subject{
		Name:        name,
		Description: req.Description,
		CreatedBy:   userRegdNo.(string),
	}

	if err := database.DB.Create(&subject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subject"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Subject created successfully",
		"subject": subject,
	})
}

// GetSubjects returns all subjects with their topics
func GetSubjects(c *gin.Context) {
	var subjects []models.Subject
	query := database.DB.Order("name ASC")

	// Optional: include topics
	if c.Query("include_topics") == "true" {
		query = query.Preload("Topics")
	}

	if err := query.Find(&subjects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subjects"})
		return
	}

	c.JSON(http.StatusOK, subjects)
}

// UpdateSubject updates an existing subject (super admin only)
func UpdateSubject(c *gin.Context) {
	id := c.Param("id")

	var subject models.Subject
	if err := database.DB.First(&subject, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subject not found"})
		return
	}

	var req CreateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject name cannot be empty"})
		return
	}

	var existing models.Subject
	if err := database.DB.Where("LOWER(name) = LOWER(?) AND id != ?", name, subject.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Subject '%s' already exists", name)})
		return
	}

	subject.Name = name
	subject.Description = req.Description

	if err := database.DB.Save(&subject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subject"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subject updated successfully",
		"subject": subject,
	})
}

// DeleteSubject deletes a subject and its topics (super admin only)
func DeleteSubject(c *gin.Context) {
	id := c.Param("id")

	var subject models.Subject
	if err := database.DB.First(&subject, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subject not found"})
		return
	}

	if err := database.DB.Delete(&subject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subject"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subject deleted successfully"})
}

// ─── Topic CRUD ──────────────────────────────────────────────────────

// CreateTopicRequest represents the request for creating a topic
type CreateTopicRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description"`
	SubjectID   uint   `json:"subject_id" binding:"required"`
}

// CreateTopic creates a new canonical topic under a subject (super admin only)
func CreateTopic(c *gin.Context) {
	var req CreateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify subject exists
	var subject models.Subject
	if err := database.DB.First(&subject, req.SubjectID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject not found"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic name cannot be empty"})
		return
	}

	// Check for duplicate topic name within the same subject
	var existing models.Topic
	if err := database.DB.Where("subject_id = ? AND LOWER(name) = LOWER(?)", req.SubjectID, name).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Topic '%s' already exists in subject '%s'", name, subject.Name)})
		return
	}

	userRegdNo, _ := c.Get("regdno")

	topic := models.Topic{
		SubjectID:   req.SubjectID,
		Name:        name,
		Description: req.Description,
		CreatedBy:   userRegdNo.(string),
	}

	if err := database.DB.Create(&topic).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create topic"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Topic created successfully",
		"topic":   topic,
	})
}

// GetTopics returns all canonical topics (accessible to all authenticated users)
// Supports ?subject_id= filter
func GetTopics(c *gin.Context) {
	var topics []models.Topic
	query := database.DB.Order("name ASC")

	// Optional subject filter
	if subjectID := c.Query("subject_id"); subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}

	if err := query.Find(&topics).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch topics"})
		return
	}

	c.JSON(http.StatusOK, topics)
}

// UpdateTopic updates an existing topic (super admin only)
func UpdateTopic(c *gin.Context) {
	id := c.Param("id")

	var topic models.Topic
	if err := database.DB.First(&topic, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	var req CreateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify subject exists if changed
	if req.SubjectID != 0 && req.SubjectID != topic.SubjectID {
		var subject models.Subject
		if err := database.DB.First(&subject, req.SubjectID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Subject not found"})
			return
		}
		topic.SubjectID = req.SubjectID
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic name cannot be empty"})
		return
	}

	var existing models.Topic
	if err := database.DB.Where("subject_id = ? AND LOWER(name) = LOWER(?) AND id != ?", topic.SubjectID, name, topic.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Topic '%s' already exists in this subject", name)})
		return
	}

	topic.Name = name
	topic.Description = req.Description

	if err := database.DB.Save(&topic).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update topic"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Topic updated successfully",
		"topic":   topic,
	})
}

// DeleteTopic deletes a topic (super admin only)
func DeleteTopic(c *gin.Context) {
	id := c.Param("id")

	var topic models.Topic
	if err := database.DB.First(&topic, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}

	if err := database.DB.Delete(&topic).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete topic"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Topic deleted successfully"})
}
