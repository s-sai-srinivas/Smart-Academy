package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CreateProblemRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Difficulty  string `json:"difficulty"`
	Tags        string `json:"tags"`
	TimeLimit   int    `json:"time_limit"`
	MemoryLimit int    `json:"memory_limit"`
}

type CreateTestCaseRequest struct {
	Input          string `json:"input" binding:"required"`
	ExpectedOutput string `json:"expected_output" binding:"required"`
	IsSample       bool   `json:"is_sample"`
	Points         int    `json:"points"`
}

func GetProblems(c *gin.Context) {
	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get user's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
		return
	}

	// Only return problems from user's college
	var problems []models.Problem
	if err := database.DB.Where("college_id = ?", *user.CollegeID).Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch problems"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"problems": problems})
}

func GetProblem(c *gin.Context) {
	id := c.Param("id")

	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get user's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
		return
	}

	// Only get problem if it belongs to user's college
	var problem models.Problem
	err := database.DB.Where("id = ? AND college_id = ?", id, *user.CollegeID).
		Preload("TestCases", "is_sample = ?", true).
		First(&problem).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"problem": problem})
}

func CreateProblem(c *gin.Context) {
	var req CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRegdNo, _ := c.Get("regdno")

	// Get user's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
		return
	}

	problem := models.Problem{
		Title:       req.Title,
		Description: req.Description,
		Difficulty:  req.Difficulty,
		Tags:        req.Tags,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		CollegeID:   user.CollegeID, // Set college ID from user
		CreatedBy:   userRegdNo.(string),
	}

	// Set defaults
	if problem.Difficulty == "" {
		problem.Difficulty = "easy"
	}
	if problem.TimeLimit == 0 {
		problem.TimeLimit = 2000
	}
	if problem.MemoryLimit == 0 {
		problem.MemoryLimit = 256000
	}

	if err := database.DB.Create(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create problem"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"problem": problem})
}

func UpdateProblem(c *gin.Context) {
	id := c.Param("id")

	// Get current user's info from context
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	role, _ := c.Get("role")

	// Get user's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
		return
	}

	// Find the problem and verify it belongs to user's college
	var problem models.Problem
	if err := database.DB.Where("id = ? AND college_id = ?", id, *user.CollegeID).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Authorization check: Only problem creator or admin/super_admin can update
	isAdmin := role == "admin" || role == "college_admin" || role == "super_admin"
	isOwner := problem.CreatedBy == userRegdNo.(string)

	if !isAdmin && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the problem creator or an admin can update this problem"})
		return
	}

	var req CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	problem.Title = req.Title
	problem.Description = req.Description
	problem.Difficulty = req.Difficulty
	problem.Tags = req.Tags
	problem.TimeLimit = req.TimeLimit
	problem.MemoryLimit = req.MemoryLimit

	if err := database.DB.Save(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"problem": problem})
}

func DeleteProblem(c *gin.Context) {
	id := c.Param("id")

	// Get current user's info from context
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	role, _ := c.Get("role")

	// Get user's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
		return
	}

	// Find the problem and verify it belongs to user's college
	var problem models.Problem
	if err := database.DB.Where("id = ? AND college_id = ?", id, *user.CollegeID).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Authorization check: Only problem creator or admin/super_admin can delete
	isAdmin := role == "admin" || role == "college_admin" || role == "super_admin"
	isOwner := problem.CreatedBy == userRegdNo.(string)

	if !isAdmin && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the problem creator or an admin can delete this problem"})
		return
	}

	if err := database.DB.Delete(&models.Problem{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Problem deleted successfully"})
}

func CreateTestCase(c *gin.Context) {
	problemID := c.Param("id")

	// Get current user's info from context
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	role, _ := c.Get("role")

	// Get user's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
		return
	}

	var req CreateTestCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify problem exists and belongs to user's college
	var problem models.Problem
	if err := database.DB.Where("id = ? AND college_id = ?", problemID, *user.CollegeID).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Authorization check: Only problem creator or admin can add test cases
	isAdmin := role == "admin" || role == "college_admin" || role == "super_admin"
	isOwner := problem.CreatedBy == userRegdNo.(string)

	if !isAdmin && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the problem creator or an admin can add test cases"})
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

func GetTestCases(c *gin.Context) {
	problemID := c.Param("id")
	role, _ := c.Get("role")

	var testCases []models.TestCase
	query := database.DB.Where("problem_id = ?", problemID)

	// Students can only see sample test cases
	if role != "admin" {
		query = query.Where("is_sample = ?", true)
	}

	if err := query.Find(&testCases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch test cases"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"test_cases": testCases})
}

func DeleteTestCase(c *gin.Context) {
	problemID := c.Param("id")
	testCaseID := c.Param("testcase_id")

	// Get current user's info from context
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	role, _ := c.Get("role")

	// Get user's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
		return
	}

	// Verify problem exists and belongs to user's college
	var problem models.Problem
	if err := database.DB.Where("id = ? AND college_id = ?", problemID, *user.CollegeID).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Authorization check: Only problem creator or admin can delete test cases
	isAdmin := role == "admin" || role == "college_admin" || role == "super_admin"
	isOwner := problem.CreatedBy == userRegdNo.(string)

	if !isAdmin && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the problem creator or an admin can delete test cases"})
		return
	}

	// Delete test case
	if err := database.DB.Delete(&models.TestCase{}, testCaseID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete test case"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test case deleted successfully"})
}
