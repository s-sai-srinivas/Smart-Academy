/**
 * SweetAlert2 utility functions for consistent error/success/warning displays
 * Usage: import { showError, showSuccess, showWarning, showInfo, showConfirm } from '@/utils/showAlert';
 */

import Swal, { type SweetAlertOptions, type SweetAlertResult } from 'sweetalert2';

// Base configuration for all alerts
const defaultConfig: SweetAlertOptions = {
  confirmButtonColor: '#f59e0b',
  cancelButtonColor: '#475569',
  background: '#1e293b',
  color: '#e2e8f0',
  customClass: {
    popup: 'swal2-custom-popup',
    title: 'swal2-custom-title',
    htmlContainer: 'swal2-custom-content',
    confirmButton: 'swal2-custom-confirm',
    cancelButton: 'swal2-custom-cancel',
  },
};

/**
 * Show a generic alert with custom options
 */
export const showAlert = (options: SweetAlertOptions): Promise<SweetAlertResult> =>
  Swal.fire({ ...defaultConfig, ...options } as SweetAlertOptions);

/**
 * Show error alert
 */
export const showError = (message: string, title = 'Error'): Promise<SweetAlertResult> =>
  Swal.fire({
    icon: 'error',
    title,
    text: message,
    ...defaultConfig,
  });

/**
 * Show success alert
 */
export const showSuccess = (message: string, title = 'Success'): Promise<SweetAlertResult> =>
  Swal.fire({
    icon: 'success',
    title,
    text: message,
    ...defaultConfig,
  });

/**
 * Show warning alert
 */
export const showWarning = (message: string, title = 'Warning'): Promise<SweetAlertResult> =>
  Swal.fire({
    icon: 'warning',
    title,
    text: message,
    ...defaultConfig,
  });

/**
 * Show info alert
 */
export const showInfo = (message: string, title = 'Info'): Promise<SweetAlertResult> =>
  Swal.fire({
    icon: 'info',
    title,
    text: message,
    ...defaultConfig,
  });

/**
 * Show confirmation dialog
 */
export const showConfirm = (
  title: string,
  text: string,
  confirmText = 'Yes',
  cancelText = 'Cancel'
): Promise<SweetAlertResult> =>
  Swal.fire({
    title,
    text,
    icon: 'question',
    showCancelButton: true,
    confirmButtonText: confirmText,
    cancelButtonText: cancelText,
    ...defaultConfig,
  });

export interface LoadingToast {
  show: (loadingMessage: string) => Promise<SweetAlertResult>;
  hide: () => void;
}

/**
 * Show loading toast (auto-closing)
 */
export const showLoading = (_message: string, _timer = 2000): LoadingToast => {
  let loadingToast: ReturnType<typeof Swal.fire> | null = null;

  const show = (loadingMessage: string): Promise<SweetAlertResult> => {
    loadingToast = Swal.fire({
      title: loadingMessage,
      allowOutsideClick: false,
      allowEscapeKey: false,
      didOpen: () => {
        Swal.showLoading();
      },
      ...defaultConfig,
    });
    return loadingToast;
  };

  const hide = (): void => {
    if (loadingToast) {
      Swal.close();
    }
  };

  return { show, hide };
};

export default {
  showAlert,
  showError,
  showSuccess,
  showWarning,
  showInfo,
  showConfirm,
  showLoading,
};
