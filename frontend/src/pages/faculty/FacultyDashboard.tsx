import { Outlet } from 'react-router-dom';

const FacultyDashboard: React.FC = () => {
  return (
    <div className="min-h-[calc(100vh-64px)] bg-background-primary">
      <main className="p-6">
        <Outlet />
      </main>
    </div>
  );
};

export default FacultyDashboard;
