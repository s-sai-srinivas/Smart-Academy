import { useEffect, type ReactNode, type MouseEvent } from 'react';

export interface ModalProps {
  children: ReactNode;
  onClose: () => void;
  title?: string;
}

function Modal({ children, onClose, title }: ModalProps) {
  const handleBackdropClick = (e: MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [onClose]);

  return (
    <div
      className="modal-overlay"
      onClick={handleBackdropClick}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <div className="modal-content relative animate-scale-in" role="document">
        <button
          className="absolute top-4 right-4 text-text-tertiary hover:text-text-primary transition-colors"
          onClick={onClose}
          aria-label="Close modal"
          type="button"
        >
          <svg
            className="w-5 h-5"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </button>
        {title && (
          <h2 id="modal-title" className="sr-only">
            {title}
          </h2>
        )}
        {children}
      </div>
    </div>
  );
}

export default Modal;
