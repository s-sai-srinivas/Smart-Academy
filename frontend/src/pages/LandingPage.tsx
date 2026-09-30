import { useState, useEffect, useRef, type ReactNode } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import {
  BookOpen,
  Code2,
  BarChart3,
  Shield,
  Users,
  GraduationCap,
  LineChart,
  ArrowRight,
  Menu,
  X,
  Zap,
  FileCheck,
  TrendingUp,
  Clock,
  Search,
  School,
  LayoutDashboard,
  Layers,
  Target,
} from 'lucide-react';

/* ─── Fade-in on scroll hook ───────────────────────────────────────── */
function useInView(threshold = 0.12) {
  const ref = useRef<HTMLDivElement | null>(null);
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const obs = new IntersectionObserver(
      ([e]) => {
        if (e.isIntersecting) setVisible(true);
      },
      { threshold }
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, [threshold]);
  return [ref, visible] as const;
}

function FadeIn({
  children,
  className = '',
  delay = 0,
}: {
  children: ReactNode;
  className?: string;
  delay?: number;
}) {
  const [ref, vis] = useInView();
  return (
    <div
      ref={ref}
      className={className}
      style={{
        opacity: vis ? 1 : 0,
        transform: vis ? 'translateY(0)' : 'translateY(20px)',
        transition: `opacity 0.6s ease ${delay}s, transform 0.6s ease ${delay}s`,
      }}
    >
      {children}
    </div>
  );
}

/* ─── Animated Counter ─────────────────────────────────────────────── */
/* ─── Analytics Hero Visual ────────────────────────────────────────── */
function AnalyticsHeroVisual() {
  const [hoveredBar, setHoveredBar] = useState<number | null>(null);

  const barData = [
    { h: 45, color: 'var(--sky-500)', label: 'Jan' },
    { h: 72, color: 'var(--sky-500)', label: 'Feb' },
    { h: 58, color: 'var(--sky-500)', label: 'Mar' },
    { h: 85, color: 'var(--amber-500)', label: 'Apr' },
    { h: 68, color: 'var(--sky-500)', label: 'May' },
    { h: 92, color: 'var(--amber-500)', label: 'Jun' },
    { h: 78, color: 'var(--sky-500)', label: 'Jul' },
  ];

  const ringProgress = 78;

  return (
    <div className="relative w-full h-[520px] lg:h-[580px] hidden lg:block overflow-hidden">
      {/* Ambient glows */}
      <div
        className="absolute top-10 right-20 w-72 h-72 rounded-full pointer-events-none opacity-30"
        style={{
          background: 'radial-gradient(circle, var(--accent-primary-strong) 0%, transparent 65%)',
        }}
      />
      <div
        className="absolute bottom-10 left-20 w-64 h-64 rounded-full pointer-events-none opacity-20"
        style={{ background: 'radial-gradient(circle, rgba(245,158,11,0.25) 0%, transparent 65%)' }}
      />

      {/* Main Analytics Dashboard Mockup */}
      <div className="absolute inset-0 flex items-center justify-center">
        <div className="relative w-[480px] h-[420px]">
          {/* Main chart card */}
          <div className="absolute top-0 left-0 w-80 h-56 rounded-2xl bg-background-secondary/80 backdrop-blur-md border border-background-border p-5 shadow-xl">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <BarChart3 className="w-4 h-4 text-accent-primary" />
                <span className="text-xs font-medium text-text-secondary">Student Performance</span>
              </div>
              <span className="text-xs text-text-muted">This Semester</span>
            </div>
            <svg viewBox="0 0 280 140" className="w-full h-32">
              {/* Grid lines */}
              {[0, 35, 70, 105, 140].map((y) => (
                <line
                  key={y}
                  x1="0"
                  y1={y}
                  x2="280"
                  y2={y}
                  stroke="rgba(148,163,184,0.1)"
                  strokeWidth="1"
                />
              ))}
              {/* Bars */}
              {barData.map((bar, i) => {
                const barWidth = 24;
                const gap = 16;
                const x = i * (barWidth + gap) + 20;
                const barHeight = bar.h * 1.3;
                const y = 140 - barHeight;
                const isHovered = hoveredBar === i;
                return (
                  <g
                    key={i}
                    style={{ cursor: 'pointer' }}
                    onMouseEnter={() => setHoveredBar(i)}
                    onMouseLeave={() => setHoveredBar(null)}
                  >
                    <rect
                      x={x}
                      y={y}
                      width={barWidth}
                      height={barHeight}
                      rx="4"
                      fill={isHovered ? bar.color : `${bar.color}80`}
                      style={{ transition: 'all 0.3s ease' }}
                    />
                    {isHovered && (
                      <text
                        x={x + barWidth / 2}
                        y={y - 8}
                        textAnchor="middle"
                        fill={bar.color}
                        fontSize="10"
                        fontWeight="600"
                      >
                        {bar.h}%
                      </text>
                    )}
                    <text
                      x={x + barWidth / 2}
                      y={155}
                      textAnchor="middle"
                      fill="rgba(148,163,184,0.5)"
                      fontSize="9"
                    >
                      {bar.label}
                    </text>
                  </g>
                );
              })}
            </svg>
          </div>

          {/* Circular progress card */}
          <div className="absolute top-4 right-0 w-40 h-40 rounded-2xl bg-background-secondary/80 backdrop-blur-md border border-background-border p-4 shadow-xl">
            <div className="flex items-center gap-2 mb-2">
              <Target className="w-3.5 h-3.5 text-sky-500" />
              <span className="text-xs font-medium text-text-secondary">Course Completion</span>
            </div>
            <div className="relative w-24 h-24 mx-auto mt-2">
              <svg viewBox="0 0 100 100" className="w-full h-full -rotate-90">
                <circle
                  cx="50"
                  cy="50"
                  r="42"
                  fill="none"
                  stroke="rgba(148,163,184,0.1)"
                  strokeWidth="8"
                />
                <circle
                  cx="50"
                  cy="50"
                  r="42"
                  fill="none"
                  stroke="var(--sky-500)"
                  strokeWidth="8"
                  strokeLinecap="round"
                  strokeDasharray={`${(2 * Math.PI * 42 * ringProgress) / 100} ${(2 * Math.PI * 42 * (100 - ringProgress)) / 100}`}
                  strokeDashoffset={0}
                  style={{ transition: 'stroke-dasharray 1.5s ease-out' }}
                />
              </svg>
              <div className="absolute inset-0 flex items-center justify-center">
                <span className="text-xl font-bold text-sky-500">{ringProgress}%</span>
              </div>
            </div>
          </div>

          {/* Line chart card */}
          <div className="absolute bottom-0 right-8 w-72 h-44 rounded-2xl bg-background-secondary/80 backdrop-blur-md border border-background-border p-5 shadow-xl">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <TrendingUp className="w-4 h-4 text-accent-secondary" />
                <span className="text-xs font-medium text-text-secondary">Weekly Submissions</span>
              </div>
            </div>
            <svg viewBox="0 0 240 100" className="w-full h-24">
              <defs>
                <linearGradient id="lineGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="var(--amber-500)" stopOpacity="0.3" />
                  <stop offset="100%" stopColor="var(--amber-500)" stopOpacity="0" />
                </linearGradient>
              </defs>
              <path
                d="M0,70 Q30,60 60,50 T120,45 T180,30 T240,20"
                fill="none"
                stroke="var(--amber-500)"
                strokeWidth="2.5"
                strokeLinecap="round"
              />
              <path
                d="M0,70 Q30,60 60,50 T120,45 T180,30 T240,20 L240,100 L0,100 Z"
                fill="url(#lineGrad)"
              />
              {/* Data points */}
              {[
                { x: 0, y: 70 },
                { x: 60, y: 50 },
                { x: 120, y: 45 },
                { x: 180, y: 30 },
                { x: 240, y: 20 },
              ].map((pt, i) => (
                <circle key={i} cx={pt.x} cy={pt.y} r="3.5" fill="var(--amber-500)" />
              ))}
            </svg>
          </div>

          {/* Mini stat pills */}
          <div className="absolute bottom-20 left-4 flex flex-col gap-2">
            <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-background-secondary/80 backdrop-blur-md border border-background-border shadow-lg">
              <div className="w-2 h-2 rounded-full bg-accent-primary" />
              <span className="text-xs text-text-secondary">Active Students</span>
              <span className="text-xs font-bold text-text-primary ml-auto">1,247</span>
            </div>
            <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-background-secondary/80 backdrop-blur-md border border-background-border shadow-lg">
              <div className="w-2 h-2 rounded-full bg-amber-500" />
              <span className="text-xs text-text-secondary">Avg Score</span>
              <span className="text-xs font-bold text-text-primary ml-auto">84.3</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ─── Timeline Step ────────────────────────────────────────────────── */
function TimelineStep({
  num,
  title,
  desc,
  icon: Icon,
  isLast,
  delay,
}: {
  num: string;
  title: string;
  desc: string;
  icon: React.ElementType;
  isLast: boolean;
  delay: number;
}) {
  const [ref, visible] = useInView(0.2);
  return (
    <div ref={ref} className="relative flex gap-6">
      {/* Timeline line */}
      {!isLast && (
        <div className="absolute left-6 top-14 w-px h-[calc(100%+2rem)] bg-gradient-to-b from-accent-primary/40 to-transparent" />
      )}

      {/* Circle node */}
      <div
        className="relative z-10 flex-shrink-0 w-12 h-12 rounded-full flex items-center justify-center border-2 transition-all duration-500"
        style={{
          borderColor: visible ? 'var(--accent-primary-strong)' : 'rgba(148,163,184,0.2)',
          backgroundColor: visible ? 'var(--accent-primary-soft)' : 'transparent',
          transform: visible ? 'scale(1)' : 'scale(0.8)',
          transitionDelay: `${delay}s`,
        }}
      >
        <span className="text-sm font-bold text-accent-primary">{num}</span>
      </div>

      {/* Content */}
      <div
        className="pb-12 transition-all duration-500"
        style={{
          opacity: visible ? 1 : 0,
          transform: visible ? 'translateX(0)' : 'translateX(-20px)',
          transitionDelay: `${delay + 0.1}s`,
        }}
      >
        <div className="flex items-center gap-3 mb-2">
          <Icon className="w-5 h-5 text-text-muted" strokeWidth={1.5} />
          <h3 className="text-xl font-semibold text-text-primary">{title}</h3>
        </div>
        <p className="text-base text-text-secondary leading-relaxed max-w-md">{desc}</p>
      </div>
    </div>
  );
}

/* ─── Horizontal Number Line ───────────────────────────────────────── */
function HorizontalNumberLine() {
  const [activeStep, setActiveStep] = useState(0);
  const [ref, visible] = useInView(0.2);

  const steps = [
    { num: 1, label: 'Setup', desc: 'Configure your college structure in minutes' },
    { num: 2, label: 'Create', desc: 'Build courses and assign lab sessions' },
    { num: 3, label: 'Learn', desc: 'Students access everything in one place' },
    { num: 4, label: 'Assess', desc: 'Automatic grading and instant feedback' },
    { num: 5, label: 'Analyze', desc: 'Track progress across all levels' },
  ];

  useEffect(() => {
    if (!visible) return;
    const interval = setInterval(() => {
      setActiveStep((prev) => (prev + 1) % steps.length);
    }, 2500);
    return () => clearInterval(interval);
  }, [visible, steps.length]);

  return (
    <div ref={ref} className="w-full">
      {/* Number line */}
      <div className="relative flex items-center justify-between mb-10">
        {/* Background line */}
        <div className="absolute left-0 right-0 top-1/2 h-0.5 bg-background-border -translate-y-1/2" />
        {/* Active progress line */}
        <div
          className="absolute left-0 top-1/2 h-0.5 bg-accent-primary -translate-y-1/2 transition-all duration-200 ease-out"
          style={{ width: `${(activeStep / (steps.length - 1)) * 100}%` }}
        />

        {steps.map((step, i) => {
          const isActive = i <= activeStep;
          const isCurrent = i === activeStep;
          return (
            <button
              key={i}
              onClick={() => setActiveStep(i)}
              className="relative z-10 flex flex-col items-center gap-3 group"
            >
              <div
                className="w-12 h-12 rounded-full flex items-center justify-center border-2 transition-all duration-500"
                style={{
                  borderColor: isActive
                    ? isCurrent
                      ? 'var(--sky-500)'
                      : 'var(--amber-500)'
                    : 'rgba(148,163,184,0.3)',
                  backgroundColor: isActive
                    ? isCurrent
                      ? 'var(--accent-primary-medium)'
                      : 'var(--accent-secondary-soft)'
                    : 'var(--bg-secondary)',
                  transform: isCurrent ? 'scale(1.15)' : 'scale(1)',
                  boxShadow: isCurrent ? '0 0 20px var(--accent-primary-strong)' : 'none',
                }}
              >
                <span
                  className="text-sm font-bold transition-colors duration-300"
                  style={{
                    color: isActive
                      ? isCurrent
                        ? 'var(--sky-500)'
                        : 'var(--amber-500)'
                      : 'var(--text-muted)',
                  }}
                >
                  {step.num}
                </span>
              </div>
              <span
                className="text-xs font-medium transition-colors duration-300"
                style={{ color: isCurrent ? 'var(--sky-500)' : 'var(--text-secondary)' }}
              >
                {step.label}
              </span>
            </button>
          );
        })}
      </div>

      {/* Active step description */}
      <div className="text-center">
        <div
          key={activeStep}
          className="inline-block px-6 py-4 rounded-xl bg-background-secondary border border-background-border"
          style={{
            animation: 'fadeSlideUp 0.4s ease-out',
          }}
        >
          <p className="text-lg text-text-primary font-medium">{steps[activeStep].desc}</p>
        </div>
      </div>

      <style>{`
        @keyframes fadeSlideUp {
          from { opacity: 0; transform: translateY(10px); }
          to { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  );
}

/* ─── Circular Role Visual ─────────────────────────────────────────── */
function CircularRoleVisual() {
  const [activeRole, setActiveRole] = useState(0);
  const [ref, _visible] = useInView(0.2);

  const roles = [
    {
      icon: Shield,
      label: 'Admin',
      color: 'var(--sky-500)',
      desc: 'Manage college setup, users, and courses from a single dashboard.',
    },
    {
      icon: LineChart,
      label: 'Principal',
      color: 'var(--amber-500)',
      desc: 'View institution-wide performance without waiting for reports.',
    },
    {
      icon: BarChart3,
      label: 'HOD',
      color: 'var(--sky-500)',
      desc: 'Oversee your department and compare section performance easily.',
    },
    {
      icon: Users,
      label: 'Faculty',
      color: 'var(--amber-500)',
      desc: 'Create content, run labs, and track students with minimal effort.',
    },
    {
      icon: GraduationCap,
      label: 'Student',
      color: 'var(--sky-500)',
      desc: 'Learn, practice, and track your own progress in one place.',
    },
  ];

  const radius = 120;
  const center = 200;

  return (
    <div ref={ref} className="relative w-full max-w-sm mx-auto aspect-square">
      <svg viewBox="0 0 400 400" className="w-full h-full">
        {/* Connecting ring */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke="var(--accent-primary-soft)"
          strokeWidth="1"
          strokeDasharray="4 4"
        />

        {/* Active arc */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke={roles[activeRole].color}
          strokeWidth="2"
          strokeLinecap="round"
          strokeDasharray={`${2 * Math.PI * radius * 0.2} ${2 * Math.PI * radius * 0.8}`}
          strokeDashoffset={-2 * Math.PI * radius * (activeRole / roles.length)}
          style={{ transition: 'all 0.5s ease' }}
          opacity="0.6"
        />

        {/* Role nodes positioned on circle */}
        {roles.map((role, i) => {
          const angle = (i * 2 * Math.PI) / roles.length - Math.PI / 2;
          const x = center + radius * Math.cos(angle);
          const y = center + radius * Math.sin(angle);
          const isActive = i === activeRole;

          return (
            <g
              key={i}
              style={{ cursor: 'pointer' }}
              onClick={() => setActiveRole(i)}
              className="transition-all duration-300"
            >
              {/* Glow effect */}
              {isActive && (
                <circle cx={x} cy={y} r="18" fill={role.color} opacity="0.15">
                  <animate attributeName="r" values="18;22;18" dur="2s" repeatCount="indefinite" />
                  <animate
                    attributeName="opacity"
                    values="0.15;0.25;0.15"
                    dur="2s"
                    repeatCount="indefinite"
                  />
                </circle>
              )}
              {/* Node circle */}
              <circle
                cx={x}
                cy={y}
                r={isActive ? 14 : 11}
                fill={isActive ? `${role.color}20` : 'var(--bg-secondary)'}
                stroke={isActive ? role.color : 'rgba(148,163,184,0.3)'}
                strokeWidth={isActive ? 2 : 1}
                style={{ transition: 'all 0.3s ease' }}
              />
              {/* Icon placeholder - using circle color indicator */}
              <circle
                cx={x}
                cy={y}
                r={isActive ? 4 : 3}
                fill={role.color}
                style={{ transition: 'all 0.3s ease' }}
              />
              {/* Label */}
              <text
                x={x + (x > center ? 32 : x < center ? -32 : 0)}
                y={y + (y > center ? 32 : y < center ? -32 : 0)}
                textAnchor={x > center ? 'start' : x < center ? 'end' : 'middle'}
                fill={isActive ? role.color : 'var(--text-secondary)'}
                fontSize="10"
                fontWeight={isActive ? '600' : '400'}
                style={{ transition: 'all 0.3s ease' }}
              >
                {role.label}
              </text>
            </g>
          );
        })}

        {/* Center info */}
        <circle
          cx={center}
          cy={center}
          r="38"
          fill="var(--bg-secondary)"
          stroke="var(--accent-primary-medium)"
          strokeWidth="1"
        />
        <text
          x={center}
          y={center - 5}
          textAnchor="middle"
          fill="var(--text-primary)"
          fontSize="10"
          fontWeight="600"
        >
          {roles[activeRole].label}
        </text>
        <text
          x={center}
          y={center + 8}
          textAnchor="middle"
          fill={roles[activeRole].color}
          fontSize="8"
          fontWeight="500"
        >
          Click to explore
        </text>
      </svg>

      {/* Description below */}
      <div className="text-center -mt-2">
        <p
          key={activeRole}
          className="text-sm text-text-secondary max-w-xs mx-auto"
          style={{ animation: 'fadeSlideUp 0.3s ease-out' }}
        >
          {roles[activeRole].desc}
        </p>
      </div>
    </div>
  );
}

/* ─── Landing Page ─────────────────────────────────────────────────── */
export default function LandingPage() {
  const { isAuthenticated, isAdmin, isSuperAdmin, isHOD, isPrincipal, isFaculty } = useAuth();
  const navigate = useNavigate();
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) return;
    if (isSuperAdmin()) navigate('/super-admin', { replace: true });
    else if (isAdmin()) navigate('/admin', { replace: true });
    else if (isHOD()) navigate('/hod', { replace: true });
    else if (isPrincipal()) navigate('/principal', { replace: true });
    else if (isFaculty()) navigate('/faculty', { replace: true });
    else navigate('/dashboard', { replace: true });
  }, [isAuthenticated, isAdmin, isFaculty, isHOD, isPrincipal, isSuperAdmin, navigate]);

  const scrollTo = (id: string) => {
    setMobileOpen(false);
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' });
  };

  const navLinks = [
    { label: 'How it works', id: 'how-it-works' },
    { label: 'Who it is for', id: 'who-it-is-for' },
  ];

  return (
    <div className="min-h-screen bg-background-primary text-text-primary font-sans selection:bg-accent-primary/30">
      {/* ═══════════════ Navbar ═══════════════ */}
      <nav className="fixed top-0 inset-x-0 z-50 bg-background-primary/80 backdrop-blur-xl border-b border-background-border">
        <div className="max-w-7xl mx-auto flex items-center justify-between px-6 lg:px-8 h-16">
          <button
            onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
            className="flex items-center gap-2.5 cursor-pointer"
          >
            <School className="w-5 h-5 text-accent-primary" strokeWidth={2} />
            <span className="text-base font-semibold tracking-tight">Smart Academy</span>
          </button>

          <div className="hidden lg:flex items-center gap-8">
            {navLinks.map((l) => (
              <button
                key={l.id}
                onClick={() => scrollTo(l.id)}
                className="text-sm text-text-secondary hover:text-text-primary transition-colors"
              >
                {l.label}
              </button>
            ))}
          </div>

          <div className="hidden lg:flex items-center gap-6">
            <Link
              to="/login"
              className="text-sm text-text-secondary hover:text-text-primary transition-colors"
            >
              Login
            </Link>
            <Link
              to="/login"
              className="text-sm font-medium text-accent-primary hover:text-sky-400 transition-colors"
            >
              Get Started
            </Link>
          </div>

          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            className="lg:hidden p-2 text-text-secondary hover:text-text-primary"
          >
            {mobileOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
        </div>

        {mobileOpen && (
          <div className="lg:hidden border-t border-background-border bg-background-primary px-6 pb-5 pt-3 space-y-1">
            {navLinks.map((l) => (
              <button
                key={l.id}
                onClick={() => scrollTo(l.id)}
                className="block w-full text-left px-3 py-2.5 text-sm text-text-secondary hover:text-text-primary transition-colors"
              >
                {l.label}
              </button>
            ))}
            <div className="pt-3 flex flex-col gap-2">
              <Link
                to="/login"
                onClick={() => setMobileOpen(false)}
                className="w-full text-center px-4 py-2.5 text-sm text-text-secondary transition-colors"
              >
                Login
              </Link>
              <Link
                to="/login"
                onClick={() => setMobileOpen(false)}
                className="w-full text-center px-4 py-2.5 text-sm font-medium text-accent-primary transition-colors"
              >
                Get Started
              </Link>
            </div>
          </div>
        )}
      </nav>

      {/* ═══════════════ Hero ═══════════════ */}
      <section className="relative min-h-screen flex flex-col justify-center pt-16 overflow-hidden">
        <div className="max-w-7xl mx-auto px-6 lg:px-8 w-full">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            <div className="max-w-2xl">
              <FadeIn>
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-accent-primary/10 border border-accent-primary/20 mb-8">
                  <div className="w-1.5 h-1.5 rounded-full bg-accent-primary animate-pulse" />
                  <span className="text-xs font-medium text-accent-primary">
                    Trusted by 50+ Engineering Colleges
                  </span>
                </div>
              </FadeIn>

              <FadeIn delay={0.1}>
                <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight leading-[1.08]">
                  One place for
                  <br />
                  <span className="text-accent-primary">courses, labs</span>
                  <br />
                  <span className="text-text-muted">and insights.</span>
                </h1>
              </FadeIn>

              <FadeIn delay={0.2}>
                <p className="mt-8 text-lg text-text-secondary leading-relaxed max-w-lg">
                  Stop juggling between spreadsheets, documents, and multiple apps. Manage your
                  entire programming department from a single dashboard.
                </p>
              </FadeIn>

              <FadeIn delay={0.3}>
                <div className="mt-10 flex flex-col sm:flex-row items-start gap-6">
                  <Link
                    to="/login"
                    className="group inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-accent-primary text-white text-sm font-semibold hover:bg-sky-600 transition-all"
                  >
                    Start Free Trial
                    <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
                  </Link>
                  <button
                    onClick={() => scrollTo('how-it-works')}
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-lg border border-background-border text-sm font-semibold text-text-secondary hover:text-text-primary hover:border-text-muted transition-all"
                  >
                    See How It Works
                  </button>
                </div>
              </FadeIn>
            </div>

            <FadeIn delay={0.2}>
              <AnalyticsHeroVisual />
            </FadeIn>
          </div>
        </div>

        {/* Bottom wave decoration */}
        <div className="absolute bottom-0 left-0 right-0">
          <svg
            viewBox="0 0 1440 100"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            className="w-full"
          >
            <path
              d="M0 50C240 90 480 10 720 50C960 90 1200 10 1440 50V100H0V50Z"
              fill="var(--bg-secondary)"
              opacity="0.5"
            />
          </svg>
        </div>
      </section>

      {/* ═══════════════ The Problem (Before/After Visual) ═══════════════ */}
      <section className="py-24 lg:py-32 bg-background-secondary">
        <div className="max-w-7xl mx-auto px-6 lg:px-8">
          <FadeIn>
            <div className="text-center max-w-2xl mx-auto mb-16">
              <h2 className="text-3xl lg:text-4xl font-bold tracking-tight leading-tight">
                Running a department should not feel like this
              </h2>
            </div>
          </FadeIn>

          <div className="grid lg:grid-cols-2 gap-12 items-center">
            {/* Before - Chaotic */}
            <FadeIn delay={0.1}>
              <div className="relative p-8 rounded-2xl bg-background-primary border border-background-border">
                <div className="absolute -top-3 left-6 px-3 py-1 rounded-full bg-amber-500/10 border border-amber-500/20">
                  <span className="text-xs font-medium text-amber-400">Before</span>
                </div>
                <div className="mt-4 space-y-4">
                  {[
                    { icon: FileCheck, text: 'Lesson plans in scattered documents' },
                    { icon: Clock, text: 'Manual grading takes days' },
                    { icon: Search, text: 'Copy-checking is nearly impossible' },
                    { icon: LayoutDashboard, text: 'No real-time view of performance' },
                  ].map((item, i) => (
                    <div
                      key={i}
                      className="flex items-center gap-3 p-3 rounded-lg bg-background-secondary/50"
                    >
                      <item.icon className="w-5 h-5 text-amber-400/70" strokeWidth={1.5} />
                      <span className="text-sm text-text-secondary">{item.text}</span>
                    </div>
                  ))}
                </div>
              </div>
            </FadeIn>

            {/* Arrow */}
            <div className="hidden lg:flex absolute left-1/2 -translate-x-1/2 items-center justify-center">
              <div className="w-12 h-12 rounded-full bg-accent-primary/10 border border-accent-primary/20 flex items-center justify-center">
                <ArrowRight className="w-5 h-5 text-accent-primary" />
              </div>
            </div>

            {/* After - Organized */}
            <FadeIn delay={0.2}>
              <div className="relative p-8 rounded-2xl bg-background-primary border border-accent-primary/20">
                <div className="absolute -top-3 left-6 px-3 py-1 rounded-full bg-sky-500/10 border border-sky-500/20">
                  <span className="text-xs font-medium text-sky-400">After Smart Academy</span>
                </div>
                <div className="mt-4 space-y-4">
                  {[
                    { icon: Layers, text: 'Theory, labs, contests & quizzes — all in one place' },
                    { icon: Zap, text: 'Code graded automatically in seconds' },
                    { icon: Shield, text: 'Copy detection with one click' },
                    { icon: BarChart3, text: 'Live dashboards for every level' },
                  ].map((item, i) => (
                    <div
                      key={i}
                      className="flex items-center gap-3 p-3 rounded-lg bg-sky-500/5 border border-sky-500/10"
                    >
                      <item.icon className="w-5 h-5 text-sky-500" strokeWidth={1.5} />
                      <span className="text-sm text-text-primary font-medium">{item.text}</span>
                    </div>
                  ))}
                </div>
              </div>
            </FadeIn>
          </div>
        </div>
      </section>

      {/* ═══════════════ How it works (Timeline) ═══════════════ */}
      <section id="how-it-works" className="py-24 lg:py-32 border-t border-background-border">
        <div className="max-w-7xl mx-auto px-6 lg:px-8">
          <FadeIn>
            <div className="text-center max-w-2xl mx-auto mb-16">
              <p className="text-sm text-text-muted mb-3 tracking-wide uppercase">How It Works</p>
              <h2 className="text-3xl lg:text-4xl font-bold tracking-tight leading-tight">
                From setup to insights in five simple steps
              </h2>
            </div>
          </FadeIn>

          {/* Horizontal Number Line for desktop */}
          <div className="hidden lg:block mb-20">
            <FadeIn>
              <HorizontalNumberLine />
            </FadeIn>
          </div>

          {/* Vertical Timeline for mobile/tablet */}
          <div className="lg:hidden max-w-md mx-auto">
            {[
              {
                num: '01',
                title: 'Set up your college',
                desc: 'Configure departments, sections, and user roles in under 10 minutes.',
                icon: School,
              },
              {
                num: '02',
                title: 'Build courses',
                desc: 'Create theory modules and lab sessions with coding problems.',
                icon: BookOpen,
              },
              {
                num: '03',
                title: 'Students learn',
                desc: 'Access course content, write code, and submit assignments.',
                icon: Code2,
              },
              {
                num: '04',
                title: 'Automatic grading',
                desc: 'Submissions are evaluated instantly against hidden test cases.',
                icon: Zap,
              },
              {
                num: '05',
                title: 'Track progress',
                desc: 'View real-time analytics from student to institution level.',
                icon: BarChart3,
              },
            ].map((step, i) => (
              <TimelineStep
                key={step.num}
                num={step.num}
                title={step.title}
                desc={step.desc}
                icon={step.icon}
                isLast={i === 4}
                delay={i * 0.1}
              />
            ))}
          </div>
        </div>
      </section>

      {/* ═══════════════ Who it is for (Circular Visual) ═══════════════ */}
      <section id="who-it-is-for" className="py-24 lg:py-32 bg-background-secondary">
        <div className="max-w-7xl mx-auto px-6 lg:px-8">
          <FadeIn>
            <div className="text-center max-w-2xl mx-auto mb-16">
              <p className="text-sm text-text-muted mb-3 tracking-wide uppercase">Who It Is For</p>
              <h2 className="text-3xl lg:text-4xl font-bold tracking-tight leading-tight">
                Built for every level of your institution
              </h2>
            </div>
          </FadeIn>

          <div className="grid lg:grid-cols-2 gap-16 items-center">
            <FadeIn delay={0.1}>
              <CircularRoleVisual />
            </FadeIn>

            <div className="space-y-6">
              {[
                {
                  icon: Shield,
                  role: 'College Admin',
                  desc: 'Set up the entire college structure and manage users from one place.',
                  color: 'var(--sky-500)',
                },
                {
                  icon: LineChart,
                  role: 'Principal',
                  desc: 'See institution-wide reports and compare departments without waiting.',
                  color: 'var(--amber-500)',
                },
                {
                  icon: BarChart3,
                  role: 'HOD',
                  desc: 'Oversee branch courses and review department-level analytics at a glance.',
                  color: 'var(--sky-500)',
                },
                {
                  icon: Users,
                  role: 'Faculty',
                  desc: 'Run lab sessions, detect copied code, and track progress with minimal effort.',
                  color: 'var(--amber-500)',
                },
                {
                  icon: GraduationCap,
                  role: 'Student',
                  desc: 'Study, practice, take quizzes, and see your own improvement over time.',
                  color: 'var(--sky-500)',
                },
              ].map((r, i) => (
                <FadeIn key={r.role} delay={i * 0.08}>
                  <div className="flex items-start gap-4 p-4 rounded-xl hover:bg-background-primary/50 transition-colors group cursor-default">
                    <div
                      className="flex-shrink-0 w-10 h-10 rounded-lg flex items-center justify-center transition-all"
                      style={{ backgroundColor: `${r.color}15` }}
                    >
                      <r.icon className="w-5 h-5" style={{ color: r.color }} strokeWidth={1.5} />
                    </div>
                    <div>
                      <h3 className="text-base font-semibold text-text-primary mb-1">{r.role}</h3>
                      <p className="text-sm text-text-secondary leading-relaxed">{r.desc}</p>
                    </div>
                  </div>
                </FadeIn>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* ═══════════════ CTA ═══════════════ */}
      <section className="py-24 lg:py-32 bg-background-secondary relative overflow-hidden">
        {/* Background decoration */}
        <div className="absolute inset-0 overflow-hidden">
          <div
            className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] rounded-full opacity-20"
            style={{
              background:
                'radial-gradient(circle, var(--accent-primary-medium) 0%, transparent 60%)',
            }}
          />
        </div>

        <div className="relative max-w-4xl mx-auto px-6 lg:px-8 text-center">
          <FadeIn>
            <h2 className="text-3xl lg:text-5xl font-bold tracking-tight leading-tight">
              Ready to simplify your
              <br />
              <span className="text-accent-primary">department management?</span>
            </h2>
            <p className="mt-6 text-lg text-text-secondary leading-relaxed max-w-xl mx-auto">
              Join colleges that have replaced scattered tools and manual work with one unified
              platform.
            </p>
            <div className="mt-10 flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link
                to="/login"
                className="group inline-flex items-center gap-2 px-8 py-4 rounded-xl bg-accent-primary text-white font-semibold hover:bg-sky-600 transition-all shadow-lg shadow-accent-primary/20"
              >
                Get Started Free
                <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
              </Link>
            </div>
            <p className="mt-6 text-sm text-text-muted">
              No credit card required. Setup takes under 10 minutes.
            </p>
          </FadeIn>
        </div>
      </section>

      {/* ═══════════════ Footer ═══════════════ */}
      <footer className="py-12 border-t border-background-border">
        <div className="max-w-7xl mx-auto px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2.5">
            <School className="w-4 h-4 text-text-muted" strokeWidth={2} />
            <span className="text-sm font-medium text-text-secondary">Smart Academy</span>
          </div>
          <p className="text-sm text-text-muted">
            &copy; {new Date().getFullYear()} Smart Academy. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
