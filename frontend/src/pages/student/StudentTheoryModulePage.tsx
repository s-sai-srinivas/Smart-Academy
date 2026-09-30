import { useState, useEffect } from 'react';
import { coursesAPI, default as apiInstance } from '../../services/api';
import { showError } from '../../utils/showAlert';
import Breadcrumb from '../../components/common/Breadcrumb';

interface ModuleItem {
  id: number;
  module_name?: string;
  title?: string;
}

interface Week {
  id: number;
  week_name: string;
  week_order?: number;
  modules?: ModuleItem[];
  has_quiz?: boolean;
}

interface TheoryPDF {
  id: number;
  theory_id: number;
  theory_module_id?: number;
  file_name: string;
  display_name?: string;
  file_size: number;
  created_at?: string;
}

interface StudentTheoryModulePageProps {
  course: { course_name: string };
  courseId: string;
  week: Week;
  module: {
    id: number;
    module_name?: string;
    title?: string;
    description?: string;
    content?: string;
  };
  theoryWeeks?: Week[];
  theoryId: number | null;
  onBack: () => void;
  onNavigateModule: (week: Week, module: ModuleItem) => void;
}

/**
 * StudentTheoryModulePage
 * Renders the content of a single theory module, including uploaded PDFs.
 * Features a collapsible sidebar showing all weeks/modules for easy navigation.
 */
function StudentTheoryModulePage({
  course,
  courseId: _courseId,
  week,
  module,
  theoryWeeks,
  theoryId,
  onBack,
  onNavigateModule,
}: StudentTheoryModulePageProps) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [pdfs, setPdfs] = useState<TheoryPDF[]>([]);
  const [pdfsLoading, setPdfsLoading] = useState(false);

  useEffect(() => {
    if (!theoryId) return;

    const fetchPDFs = async () => {
      setPdfsLoading(true);
      try {
        const response = (await coursesAPI.getTheoryPDFs(theoryId)) as { pdfs?: TheoryPDF[] };
        setPdfs(response.pdfs || []);
      } catch (err) {
        console.error('Failed to load PDFs:', err);
        showError('Failed to load PDFs');
      } finally {
        setPdfsLoading(false);
      }
    };

    fetchPDFs();
  }, [theoryId]);

  // Filter PDFs for this specific module vs general course PDFs
  const modulePdfs = pdfs.filter((pdf) => pdf.theory_module_id === module.id);

  const renderPdfList = (items: TheoryPDF[], emptyMessage: string) => {
    if (items.length === 0) {
      return <p className="text-text-muted text-sm italic">{emptyMessage}</p>;
    }

    return (
      <div className="space-y-3">
        {items.map((pdf) => (
          <button
            key={pdf.id}
            type="button"
            onClick={() => handlePdfOpen(pdf.id)}
            className="w-full flex items-center gap-3 p-3 bg-background-tertiary rounded-lg border border-background-border hover:border-accent-secondary/50 hover:bg-background-elevated transition-all text-left"
          >
            <div className="w-10 h-10 rounded-lg bg-red-500/10 flex items-center justify-center text-red-400 flex-shrink-0">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
                />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-text-primary text-sm font-medium truncate">
                {pdf.display_name || pdf.file_name}
              </p>
              <p className="text-text-muted text-xs">{(pdf.file_size / 1024).toFixed(1)} KB</p>
            </div>
            <svg
              className="w-4 h-4 text-text-muted"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
              />
            </svg>
          </button>
        ))}
      </div>
    );
  };

  const handlePdfOpen = async (pdfId: number) => {
    try {
      const blob = (await apiInstance.get(`/theory-pdfs/${pdfId}`, {
        responseType: 'blob',
      })) as Blob;
      const url = window.URL.createObjectURL(blob);
      window.open(url, '_blank');
      // Clean up the blob URL after a delay
      setTimeout(() => window.URL.revokeObjectURL(url), 120000);
    } catch (err) {
      console.error('Failed to open PDF:', err);
      showError('Failed to open PDF');
    }
  };

  return (
    <div className="flex min-h-screen">
      {/* Collapsible Sidebar */}
      <div
        className={`${sidebarCollapsed ? 'w-12' : 'w-72'} flex-shrink-0 border-r border-border-light bg-bg-primary transition-all duration-300`}
      >
        {/* Sidebar Header */}
        <div className="flex items-center justify-between p-3 border-b border-border-light">
          {!sidebarCollapsed && (
            <span className="text-sm font-semibold text-text-primary truncate">
              {course.course_name}
            </span>
          )}
          <button
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            className="p-1.5 rounded hover:bg-bg-secondary text-text-muted hover:text-text-primary transition-colors"
            title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {sidebarCollapsed ? (
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 5l7 7-7 7"
                />
              </svg>
            ) : (
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 19l-7-7 7-7"
                />
              </svg>
            )}
          </button>
        </div>

        {/* Weeks/Modules List */}
        {!sidebarCollapsed && (
          <div className="overflow-y-auto max-h-[calc(100vh-60px)]">
            {theoryWeeks?.map((w, wIndex) => (
              <div key={w.id} className="border-b border-border-light/50">
                {/* Week Header */}
                <div
                  className={`px-3 py-2 text-xs font-semibold uppercase tracking-wide
                  ${w.id === week.id ? 'text-accent-secondary bg-accent-secondary/5' : 'text-text-muted bg-bg-secondary'}`}
                >
                  Week {w.week_order || wIndex + 1}: {w.week_name}
                </div>
                {/* Modules */}
                {w.modules?.map((m) => (
                  <button
                    key={m.id}
                    onClick={() => onNavigateModule(w, m)}
                    className={`w-full text-left px-3 py-2 text-sm transition-colors flex items-center gap-2
                      ${
                        m.id === module.id
                          ? 'bg-accent-secondary/10 text-accent-secondary font-medium border-l-2 border-accent-secondary'
                          : 'text-text-secondary hover:bg-bg-secondary hover:text-text-primary border-l-2 border-transparent'
                      }`}
                  >
                    <span className="text-xs">📄</span>
                    <span className="truncate">{m.module_name || m.title}</span>
                  </button>
                ))}
                {/* Quiz Link */}
                {w.has_quiz && (
                  <button
                    onClick={() => onBack()}
                    className="w-full text-left px-3 py-2 text-sm text-text-muted hover:text-accent-secondary hover:bg-bg-secondary transition-colors flex items-center gap-2 border-l-2 border-transparent"
                  >
                    <span className="text-xs">📝</span>
                    <span className="truncate">Practice Quiz</span>
                  </button>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Collapsed view - just icons */}
        {sidebarCollapsed && (
          <div className="flex flex-col items-center py-2 gap-1">
            {theoryWeeks?.map((w, wIndex) => (
              <div key={w.id} className="relative group">
                <div
                  className={`w-8 h-8 rounded flex items-center justify-center text-xs font-medium
                  ${w.id === week.id ? 'bg-accent-secondary text-white' : 'bg-bg-secondary text-text-muted hover:text-text-primary'}`}
                >
                  {w.week_order || wIndex + 1}
                </div>
                {/* Tooltip */}
                <div className="absolute left-full ml-2 px-2 py-1 bg-bg-elevated border border-border-light rounded text-xs text-text-primary whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity z-10 pointer-events-none">
                  Week {w.week_order || wIndex + 1}: {w.week_name}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Main Content */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-6 max-w-4xl mx-auto">
          {/* Breadcrumb */}
          <Breadcrumb
            items={[
              { label: 'Dashboard', to: '/dashboard' },
              { label: 'Courses', to: '/courses' },
              { label: course.course_name, onClick: onBack },
              { label: week.week_name, onClick: onBack },
              { label: module.module_name || module.title || 'Module' },
            ]}
          />

          {/* Module Header */}
          <div className="card mb-6">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-xl bg-accent-primary/10 flex items-center justify-center text-2xl flex-shrink-0">
                📄
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-1">
                  <span className="text-xs font-medium text-accent-secondary uppercase tracking-wide">
                    Week {week.week_order}: {week.week_name}
                  </span>
                </div>
                <h1 className="text-2xl font-bold text-text-primary mb-2">
                  {module.module_name || module.title || 'Module'}
                </h1>
                {module.description && (
                  <p className="text-text-secondary text-sm leading-relaxed">
                    {module.description}
                  </p>
                )}
              </div>
            </div>
          </div>

          <div className="card">
            {module.content ? (
              <div className="prose-style">
                <ModuleContent content={module.content} />
              </div>
            ) : (
              <div className="text-center py-16">
                <div className="text-5xl mb-4">📚</div>
                <h3 className="text-lg font-semibold text-text-primary mb-2">
                  Content Coming Soon
                </h3>
                <p className="text-text-secondary text-sm">
                  The content for this module is being prepared.
                </p>
              </div>
            )}
          </div>

          <div className="card mt-6">
            <h2 className="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
              <span>📎</span>
              Module PDFs
            </h2>
            {renderPdfList(modulePdfs, 'No module PDFs attached yet.')}
          </div>

          {/* PDF loading state */}
          {pdfsLoading && (
            <div className="card mt-6 text-center py-8">
              <div className="inline-block animate-spin rounded-full h-5 w-5 border-b-2 border-accent-primary"></div>
              <p className="text-text-secondary text-sm mt-2">Loading PDFs...</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/**
 * Renders module content — handles plain text and basic markdown-like formatting.
 * If `content` is valid JSON (array of blocks), renders structured content.
 * Otherwise renders as a formatted text block.
 */
function ModuleContent({ content }: { content: string }) {
  // Try to parse as JSON (structured content blocks from AI generation)
  let parsed: unknown = null;
  try {
    parsed = JSON.parse(content);
  } catch {
    // Not JSON — render as plain/rich text
  }

  if (parsed && Array.isArray(parsed)) {
    return (
      <div className="space-y-6">
        {parsed.map((block, i) => (
          <ContentBlock key={i} block={block} />
        ))}
      </div>
    );
  }

  // Fallback: render as preformatted / paragraph text
  return (
    <div className="text-text-secondary leading-relaxed whitespace-pre-wrap text-sm">{content}</div>
  );
}

interface ContentBlockData {
  type?: string;
  text?: string;
  content?: string;
  language?: string;
  code?: string;
  items?: Array<string | { text?: string }>;
  points?: Array<string | { text?: string }>;
  description?: string;
}

function ContentBlock({ block }: { block: unknown }) {
  if (!block || typeof block !== 'object') {
    return <p className="text-text-secondary text-sm">{String(block)}</p>;
  }

  const b = block as ContentBlockData;

  switch (b.type) {
    case 'heading':
      return (
        <h2 className="text-xl font-bold text-text-primary border-b border-background-border pb-2">
          {b.text || b.content}
        </h2>
      );
    case 'subheading':
      return <h3 className="text-lg font-semibold text-text-primary">{b.text || b.content}</h3>;
    case 'paragraph':
    case 'text':
      return <p className="text-text-secondary text-sm leading-relaxed">{b.text || b.content}</p>;
    case 'bullet_points':
    case 'list':
      return (
        <ul className="space-y-1.5 pl-4">
          {(b.items || b.points || []).map((item, i) => (
            <li key={i} className="flex items-start gap-2 text-text-secondary text-sm">
              <span className="text-accent-secondary mt-1 flex-shrink-0">•</span>
              <span>{typeof item === 'string' ? item : item.text || JSON.stringify(item)}</span>
            </li>
          ))}
        </ul>
      );
    case 'code':
      return (
        <div className="rounded-lg overflow-hidden border border-background-border">
          <div className="px-4 py-2 bg-background-tertiary border-b border-background-border flex items-center justify-between">
            <span className="text-xs font-mono text-text-tertiary">{b.language || 'code'}</span>
          </div>
          <pre className="p-4 bg-background-secondary overflow-x-auto text-sm font-mono text-text-primary">
            <code>{b.code || b.content}</code>
          </pre>
        </div>
      );
    case 'note':
    case 'tip':
    case 'info':
      return (
        <div className="flex gap-3 p-4 rounded-lg bg-accent-secondary/5 border border-accent-secondary/20">
          <span className="text-lg flex-shrink-0">💡</span>
          <p className="text-text-secondary text-sm leading-relaxed">{b.text || b.content}</p>
        </div>
      );
    default: {
      const text = b.text || b.content || b.description;
      if (text) {
        return <p className="text-text-secondary text-sm leading-relaxed">{text}</p>;
      }
      return null;
    }
  }
}

export default StudentTheoryModulePage;
