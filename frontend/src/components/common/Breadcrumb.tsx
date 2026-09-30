import { Link, useNavigate } from 'react-router-dom';
import { type ReactNode } from 'react';
import './Breadcrumb.css';

export interface BreadcrumbItem {
  label: string;
  to?: string;
  onClick?: () => void;
}

export interface BreadcrumbProps {
  items?: BreadcrumbItem[];
  actions?: ReactNode;
  backToPrevious?: boolean;
}

function Breadcrumb({ items = [], actions, backToPrevious = false }: BreadcrumbProps) {
  const navigate = useNavigate();

  const handleBack = () => {
    if (backToPrevious) {
      navigate(-1);
      return;
    }
    const backTarget = [...items].reverse().find((item, idx) => idx > 0 && item.to);
    if (backTarget?.to) {
      navigate(backTarget.to);
    } else {
      navigate(-1);
    }
  };

  return (
    <nav className="breadcrumb-nav" aria-label="Breadcrumb">
      <button
        className="breadcrumb-back-btn"
        onClick={handleBack}
        title="Go back"
        aria-label="Go back"
      >
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <polyline points="15 18 9 12 15 6"></polyline>
        </svg>
      </button>

      <div className="breadcrumb-trail">
        {items.map((item, index) => {
          const isLast = index === items.length - 1;
          return (
            <span
              key={index}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '0.375rem' }}
            >
              {index > 0 && (
                <span className="breadcrumb-separator" aria-hidden="true">
                  ›
                </span>
              )}
              {isLast || (!item.to && !item.onClick) ? (
                <span className="breadcrumb-current" title={item.label}>
                  {item.label}
                </span>
              ) : item.to ? (
                <Link to={item.to} className="breadcrumb-link">
                  {item.label}
                </Link>
              ) : (
                <button
                  onClick={item.onClick}
                  className="breadcrumb-link"
                  style={{
                    background: 'none',
                    border: 'none',
                    padding: 0,
                    cursor: 'pointer',
                    font: 'inherit',
                    color: 'inherit',
                  }}
                >
                  {item.label}
                </button>
              )}
            </span>
          );
        })}
      </div>

      {actions && (
        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          {actions}
        </div>
      )}
    </nav>
  );
}

export default Breadcrumb;
