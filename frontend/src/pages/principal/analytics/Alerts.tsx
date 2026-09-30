import { useState, useEffect } from 'react';
import { principalAnalyticsAPI } from '../../../services/api';
import { showError } from '../../../utils/showAlert';
import { AlertTriangle, CheckCircle2, Info, Filter, Bell, AlertOctagon } from 'lucide-react';

interface AlertItem {
  type: string;
  message: string;
  severity: string;
  branch_name?: string;
  created_at: string;
}

const Alerts = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [alerts, setAlerts] = useState<AlertItem[]>([]);
  const [filter, setFilter] = useState<string>('all');

  useEffect(() => {
    fetchAlerts();
  }, []);

  const fetchAlerts = async () => {
    try {
      const response = (await principalAnalyticsAPI.getAlerts()) as { alerts: AlertItem[] };
      setAlerts(response.alerts);
    } catch (error) {
      console.error('Error fetching alerts:', error);
      showError('Failed to load alerts');
    } finally {
      setLoading(false);
    }
  };

  const filteredAlerts = alerts.filter((alert) => {
    if (filter === 'all') return true;
    return alert.severity === filter;
  });

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'critical':
        return <AlertOctagon className="w-5 h-5 text-red-400" />;
      case 'warning':
        return <AlertTriangle className="w-5 h-5 text-yellow-400" />;
      case 'info':
        return <Info className="w-5 h-5 text-blue-400" />;
      default:
        return <Bell className="w-5 h-5 text-text-muted" />;
    }
  };

  const getSeverityBadge = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-500/20 text-red-400 border-red-500/30';
      case 'warning':
        return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30';
      case 'info':
        return 'bg-blue-500/20 text-blue-400 border-blue-500/30';
      default:
        return 'bg-gray-500/20 text-gray-400 border-gray-500/30';
    }
  };

  const getAlertTypeLabel = (type: string) => {
    return type.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase());
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex items-center gap-3">
          <svg
            className="animate-spin h-5 w-5 text-accent-primary"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            ></circle>
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          <span className="text-text-secondary">Loading alerts...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-text-primary mb-1">Risk Alerts</h1>
          <p className="text-text-secondary text-sm">
            {alerts.length} active alerts requiring attention
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-text-muted" />
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="bg-background-tertiary border border-background-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent-primary"
          >
            <option value="all">All Severities</option>
            <option value="critical">Critical</option>
            <option value="warning">Warning</option>
            <option value="info">Info</option>
          </select>
        </div>
      </div>

      {/* Alert Summary */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="card-elevated border-l-4 border-red-500">
          <div className="flex items-center gap-3">
            <AlertOctagon className="w-6 h-6 text-red-400" />
            <div>
              <div className="text-2xl font-bold text-text-primary">
                {alerts.filter((a) => a.severity === 'critical').length}
              </div>
              <div className="text-sm text-text-secondary">Critical</div>
            </div>
          </div>
        </div>
        <div className="card-elevated border-l-4 border-yellow-500">
          <div className="flex items-center gap-3">
            <AlertTriangle className="w-6 h-6 text-yellow-400" />
            <div>
              <div className="text-2xl font-bold text-text-primary">
                {alerts.filter((a) => a.severity === 'warning').length}
              </div>
              <div className="text-sm text-text-secondary">Warnings</div>
            </div>
          </div>
        </div>
        <div className="card-elevated border-l-4 border-blue-500">
          <div className="flex items-center gap-3">
            <Info className="w-6 h-6 text-blue-400" />
            <div>
              <div className="text-2xl font-bold text-text-primary">
                {alerts.filter((a) => a.severity === 'info').length}
              </div>
              <div className="text-sm text-text-secondary">Info</div>
            </div>
          </div>
        </div>
      </div>

      {/* Alert List */}
      {filteredAlerts.length > 0 ? (
        <div className="space-y-3">
          {filteredAlerts.map((alert, index) => (
            <div key={index} className="card hover:shadow-md transition-shadow">
              <div className="flex items-start gap-4">
                {getSeverityIcon(alert.severity)}
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <span
                      className={`px-2 py-0.5 rounded-full text-xs border ${getSeverityBadge(
                        alert.severity
                      )}`}
                    >
                      {alert.severity.toUpperCase()}
                    </span>
                    <span className="text-xs text-text-muted">{getAlertTypeLabel(alert.type)}</span>
                  </div>
                  <p className="text-text-primary">{alert.message}</p>
                  {alert.branch_name && (
                    <p className="text-text-muted text-sm mt-1">Branch: {alert.branch_name}</p>
                  )}
                  <p className="text-text-muted text-xs mt-2">{alert.created_at}</p>
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="card text-center py-12">
          <CheckCircle2 className="w-12 h-12 text-accent-success mx-auto mb-4" />
          <p className="text-text-secondary">No alerts match the selected filter.</p>
          {filter !== 'all' && (
            <button
              onClick={() => setFilter('all')}
              className="mt-2 text-accent-primary hover:underline text-sm"
            >
              Show all alerts
            </button>
          )}
        </div>
      )}
    </div>
  );
};

export default Alerts;
