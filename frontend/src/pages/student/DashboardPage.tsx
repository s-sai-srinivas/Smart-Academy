import { useAuth } from '../../context/AuthContext';
import StudentDashboard from './StudentDashboard';
import FacultyDashboard from '../faculty/FacultyDashboard';

function DashboardPage() {
  const { user } = useAuth();

  return (
    <div className="min-h-[calc(100vh-64px)] bg-background-primary">
      {user?.role === 'faculty' ? <FacultyDashboard /> : <StudentDashboard />}
    </div>
  );
}

export default DashboardPage;
