# Smart Academy

Production-oriented coding education platform for colleges and universities. It combines course/lab management, secure code execution, plagiarism detection, AI-assisted content generation, role-based administration, and analytics in a single system.

## What Is New

### AI Course Generation (Lesson Plan Agent)
- Upload lesson plans (PDF/DOCX/TXT) and process them asynchronously.
- AI extracts week/module structure from document content.
- Admin review workflow before publishing generated content.
- Embedded week-level practice quizzes with regenerate and edit support.
- Background generation jobs with status polling and preview endpoints.

### AI Quiz System (End-to-End)
- Faculty can generate quiz questions from theory modules using AI.
- Supports MCQ, True/False, Multi-select, Short Answer, and Fill in the Blank.
- Difficulty mix controls and Bloom level tagging.
- Full quiz lifecycle: draft, publish, close, archive.
- Student attempt flow with timer, auto-submit, result breakdown, and explanations.
- Faculty and HOD analytics views for quiz performance.

### AI Contest Generation (Multi-Agent)
- Admin can generate contest-ready problems from selected topics.
- Agentic multi-agent generation flow with verification-oriented review.
- Async job execution and approval flow for generated problems.
- Contest editorial generation and retrieval endpoints.

## Multi-Agent Workflows

### Course Generation Workflow (Lesson Plan to Course Structure)
1. Admin uploads a lesson plan document for a target course.
2. A generation job is created and processed asynchronously.
3. Multi-agent orchestration runs with role-specialized agents:
	- Extractor agent reads document chunks/tables and identifies key topics.
	- Architect agent drafts week/module structure and validates coverage.
	- Writer agent expands modules into learning content and formatting.
	- Quizmaster agent generates week-level practice quiz questions.
	- Critic agent validates schema and content quality, then suggests corrections.
4. The system stores generated output as preview data for human review.
5. Admin reviews, edits, and approves.
6. Approved data is persisted as theory weeks/modules and embedded practice quizzes.

### Contest Generation Workflow (Agentic Problem Pipeline)
1. Admin submits topics, difficulty, and requested problem count.
2. A contest generation job is started and tracked via job status endpoints.
3. The contest service invokes the orchestrator's ReAct-style agent loop.
4. The contest agent produces problem statement, constraints, test cases, and reference solution.
5. Critic/verification steps check structure, quality, and solvability.
6. Service-level verification executes reference solutions against test cases.
7. Generated problems are returned for review with verification status.
8. Admin approves selected problems, optionally attaches them to a contest, and publishes.

Note: if the orchestrator is unavailable, the service falls back to sequential AI generation to preserve availability.

### Expanded Institutional Role Model
- Roles: Super Admin, Admin, HOD, Principal, Faculty, Student.
- Super Admin college lifecycle management.
- Principal HOD assignment/removal controls.
- HOD faculty assignment, branch analytics, and quiz oversight.

### Security and Platform Hardening
- JWT authentication with role middleware and college isolation.
- Logout with token revocation support.
- Password change and admin password reset routes with rate limiting.
- Global and endpoint-specific rate limiters (login, submit, run, contest, password reset).
- Request body size limiting and structured recovery middleware.
- Auth-protected code execution route.

### Platform and Performance Enhancements
- Redis/Dragonfly cache support with in-memory fallback.
- Solve session start/end tracking for more accurate coding time telemetry.
- Contest participation and leaderboard workflow.
- Practice quiz retrieval for theory weeks across authenticated users.

## Core Capabilities

### Academic Management
- Curriculum management and course mapping.
- Separate Lab and Theory course flows.
- Theory organization with weeks and modules.
- Lab sessions with problem assignment.

### Coding Workflow
- Problem bank with topic-level problem listing.
- Run and submit flow for coding problems.
- Judge0-backed execution with resource constraints.
- Submission history, stats, and code retrieval endpoints.

### Plagiarism Detection
- Problem-level and submission-level plagiarism checks.
- Result retrieval endpoints for review and reporting.

### Dashboards and Analytics
- Student dashboard and progress surfaces.
- Faculty course/section/student analytics.
- Admin dashboard statistics.
- HOD branch/course analytics and faculty oversight.
- Principal institutional dashboard.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go, Gin |
| ORM/DB | GORM, PostgreSQL |
| Frontend | React, Vite, React Router |
| UI/Charts | Tailwind CSS, Recharts, Lucide |
| Code Editor | Monaco Editor |
| Code Execution | Judge0 |
| Plagiarism | JPlag |
| Cache | Redis/Dragonfly with in-memory fallback |

## Project Structure

```text
coding-platform/
├── backend/
│   ├── main.go
│   ├── handlers/
│   ├── middleware/
│   ├── services/
│   ├── models/
│   ├── database/
│   └── migrations/
├── frontend/
│   ├── src/
│   ├── public/
│   └── package.json
├── deployment/
└── docs/
```

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- PostgreSQL
- Judge0 service (for code execution)
- Optional: Redis/Dragonfly (for cache)

### 1. Backend Setup

```bash
cd backend
go mod tidy
cp .env.example .env
# edit .env with database/JWT/AI/Judge0 settings
go run .
```

Backend health check:

```bash
curl http://localhost:8080/health
```

### 2. Frontend Setup

```bash
cd frontend
npm install
npm run dev
```

Default frontend URL: `http://localhost:5173`

## Important Environment Variables (Backend)

| Variable | Description | Required |
|----------|-------------|----------|
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | PostgreSQL connection | `DB_PASSWORD` required |
| `JWT_SECRET` | JWT signing secret | Yes |
| `JWT_ISSUER`, `JWT_AUDIENCE` | JWT claims configuration | No |
| `JUDGE0_URL` | Judge0 base URL | Yes for run/submit |
| `AI_PROVIDER`, `AI_API_KEY`, `AI_MODEL`, `AI_MAX_TOKENS`, `AI_TEMPERATURE`, `AI_RATE_LIMIT_RPM` | AI features (quiz/lesson plan/contest generation) | AI key required for AI features |
| `REDIS_ENABLED`, `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB` | Cache configuration | Optional |
| `CORS_ALLOWED_ORIGINS` | Comma-separated allowed origins | Recommended |
| `SUPER_ADMIN_REGDNO`, `SUPER_ADMIN_EMAIL`, `SUPER_ADMIN_PASSWORD` | Super admin bootstrap | Optional but recommended |
| `JPLAG_BASE_DIR`, `JPLAG_SUBMISSIONS_DIR`, `JPLAG_RESULTS_DIR`, `JPLAG_DOCKER_IMAGE`, `JPLAG_TIMEOUT_SECONDS` | Plagiarism service settings | Needed for plagiarism checks |

## API Surface (High-Level)

Base API prefix: `/api`

### Public
- `POST /api/auth/login`
- `GET /api/colleges`

### Authenticated Core
- Profile and auth management (`/api/me`, `/api/auth/logout`, password change)
- Problems, submissions, run/submit, solve sessions
- Courses, topics, dashboards
- Student quizzes and contests

### Admin
- Curriculum, course, lab, theory, and onboarding management
- Lesson plan generation and approval workflow
- Contest generation, review, approval, editorial generation
- Plagiarism checks and result retrieval

### Faculty
- Quiz creation/generation/editing/publishing/analytics
- Course and student analytics

### HOD and Principal
- HOD analytics/faculty assignment/quiz oversight
- Principal dashboard and HOD management

### Super Admin
- College lifecycle and status management

## Frontend Modules (Implemented)

- Student: dashboard, courses, coding workspace, contests, quizzes, profile
- Faculty: dashboards, analytics, contest views, full quiz management
- Admin: dashboard, courses/labs/theory, onboarding, contests, generated content review
- HOD: analytics, faculty management, course oversight
- Principal: dashboard and HOD management
- Super Admin: college-level management dashboard

## Notes

- Database migrations run automatically on backend startup.
- AI features depend on valid provider credentials.
- Redis is optional; the app falls back to in-memory cache when disabled/unavailable.



