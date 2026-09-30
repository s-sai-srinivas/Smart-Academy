import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { facultyAnalyticsAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import StudentProfileCard from '../../components/common/StudentProfileCard';
import type { StudentAnalytics } from '../../components/common/StudentProfileCard';

/**
 * Faculty Student Analytics Page
 *
 * Standalone routed page at /faculty/student/:id/analytics
 * Fetches student data via faculty API and renders the shared StudentProfileCard.
 */
function StudentAnalyticsPage() {
  const { id } = useParams<{ id: string }>();
  const numericId = Number(id);
  const navigate = useNavigate();
  const [loading, setLoading] = useState<boolean>(true);
  const [studentData, setStudentData] = useState<StudentAnalytics | null>(null);
  const [studentInfo, setStudentInfo] = useState<{
    name?: string;
    roll_number?: string;
    section_name?: string;
  } | null>(null);

  const load = useCallback(async (): Promise<void> => {
    try {
      const response = (await facultyAnalyticsAPI.getStudentAnalytics(
        numericId
      )) as StudentAnalytics & {
        student?: { name?: string; roll_number?: string; section_name?: string };
      };
      setStudentData(response);
      setStudentInfo(response.student || null);
    } catch (err) {
      console.error('Failed to load student analytics:', err);
      showError('Failed to load student analytics');
    } finally {
      setLoading(false);
    }
  }, [numericId]);

  useEffect(() => {
    load();
  }, [load]);

  if (loading) {
    return (
      <div className="p-6">
        <div className="card text-center py-12">
          <p className="text-text-muted">Loading student analytics...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <StudentProfileCard
        studentAnalytics={studentData as StudentAnalytics}
        studentInfo={{
          name: studentInfo?.name,
          roll_number: studentInfo?.roll_number,
          section_name: studentInfo?.section_name,
        }}
        loading={loading}
        onBack={() => navigate(-1)}
        backLabel="← Back to Students"
      />
    </div>
  );
}

export default StudentAnalyticsPage;
