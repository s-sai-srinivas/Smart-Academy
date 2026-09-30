import { useState, useEffect, useMemo } from 'react';
import { principalAPI } from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';

interface HODInfo {
  name: string;
  regdno: string;
}

interface AvailableFaculty {
  regdno: string;
  name: string;
  designation?: string;
}

interface Department {
  program_id: number | string;
  program_name: string;
  program_code: string;
  branch_id: number | string;
  branch_name: string;
  current_hod: HODInfo | null;
  available_faculty: AvailableFaculty[];
}

interface Program {
  id: number | string;
  name: string;
  code: string;
}

const HODManagement = () => {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [actionLoading, setActionLoading] = useState<number | string | null>(null);
  const [selectedFaculty, setSelectedFaculty] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string>('');
  const [errorMessage, setErrorMessage] = useState<string>('');
  const [activeProgram, setActiveProgram] = useState<number | string | null>(null);
  const [activeBranch, setActiveBranch] = useState<number | string | null>(null);

  useEffect(() => {
    fetchDepartments();
  }, []);

  // Auto-clear alerts after 4s
  useEffect(() => {
    if (successMessage) {
      const t = setTimeout(() => setSuccessMessage(''), 4000);
      return () => clearTimeout(t);
    }
  }, [successMessage]);
  useEffect(() => {
    if (errorMessage) {
      const t = setTimeout(() => setErrorMessage(''), 4000);
      return () => clearTimeout(t);
    }
  }, [errorMessage]);

  const fetchDepartments = async () => {
    try {
      const response = (await principalAPI.getDepartments()) as Department[];
      const data = response || [];
      setDepartments(data);

      // Auto-select first program + branch
      if (data.length > 0) {
        const programs = [...new Map(data.map((d) => [d.program_id, d])).values()];
        const firstProg = programs[0];
        setActiveProgram(firstProg.program_id);
        const firstBranch = data.find((d) => d.program_id === firstProg.program_id);
        if (firstBranch) setActiveBranch(firstBranch.branch_id);
      }
    } catch (error) {
      console.error('Error fetching departments:', error);
      showError('Failed to load departments');
    } finally {
      setLoading(false);
    }
  };

  // Derive programs and branches from departments
  const programs = useMemo<Program[]>(() => {
    const map = new Map<number | string, Program>();
    departments.forEach((d) => {
      if (!map.has(d.program_id)) {
        map.set(d.program_id, { id: d.program_id, name: d.program_name, code: d.program_code });
      }
    });
    return [...map.values()];
  }, [departments]);

  const branches = useMemo<Department[]>(() => {
    return departments.filter((d) => d.program_id === activeProgram);
  }, [departments, activeProgram]);

  const activeDept = useMemo<Department | null>(() => {
    return departments.find((d) => d.branch_id === activeBranch) || null;
  }, [departments, activeBranch]);

  const handleProgramChange = (programId: number | string) => {
    setActiveProgram(programId);
    setSelectedFaculty(null);
    const firstBranch = departments.find((d) => d.program_id === programId);
    setActiveBranch(firstBranch ? firstBranch.branch_id : null);
  };

  const handleBranchChange = (branchId: number | string) => {
    setActiveBranch(branchId);
    setSelectedFaculty(null);
  };

  const handleAssignHOD = async () => {
    if (!selectedFaculty || !activeBranch) {
      showError('Select a faculty member first');
      return;
    }

    setActionLoading(activeBranch);
    setSuccessMessage('');
    setErrorMessage('');

    try {
      const response = (await principalAPI.assignHOD(activeBranch, selectedFaculty)) as {
        message?: string;
      };
      showSuccess(response.message || 'HOD assigned successfully');
      setSelectedFaculty(null);
      await fetchDepartments();
    } catch (error) {
      const err = error as { message?: string };
      showError(err.message || 'Failed to assign HOD');
    } finally {
      setActionLoading(null);
    }
  };

  const handleRemoveHOD = async () => {
    if (!activeDept?.current_hod) return;
    if (
      !window.confirm(
        `Remove ${activeDept.current_hod.name} as HOD? They will be demoted to faculty.`
      )
    )
      return;

    setActionLoading(activeBranch);
    setSuccessMessage('');
    setErrorMessage('');

    try {
      if (!activeBranch) return;
      const response = (await principalAPI.removeHOD(activeBranch)) as { message?: string };
      showSuccess(response.message || 'HOD removed successfully');
      await fetchDepartments();
    } catch (error) {
      const err = error as { message?: string };
      showError(err.message || 'Failed to remove HOD');
    } finally {
      setActionLoading(null);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex items-center gap-3">
          <svg
            className="animate-spin h-5 w-5 text-text-tertiary"
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
          <span className="text-text-secondary text-sm">Loading departments...</span>
        </div>
      </div>
    );
  }

  if (departments.length === 0) {
    return (
      <div className="space-y-4">
        <h1 className="text-xl font-semibold text-text-primary">HOD Management</h1>
        <div className="py-16 text-center">
          <p className="text-text-secondary">No departments found in your college.</p>
          <p className="text-text-tertiary text-sm mt-1">Faculty must be onboarded first.</p>
        </div>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-lg font-semibold text-text-primary tracking-tight">HOD Management</h1>
        <p className="text-text-tertiary text-xs mt-0.5">
          Assign or update Head of Department for each branch
        </p>
      </div>

      {/* Alerts */}
      {successMessage && (
        <div className="mb-4 px-3 py-2 text-xs rounded-[6px] bg-accent-success/8 border border-accent-success/20 text-accent-success">
          {successMessage}
        </div>
      )}
      {errorMessage && (
        <div className="mb-4 px-3 py-2 text-xs rounded-[6px] bg-accent-danger/8 border border-accent-danger/20 text-accent-danger">
          {errorMessage}
        </div>
      )}

      {/* Program Tabs */}
      <div className="mb-5">
        <div className="flex gap-0 border-b border-background-border/60">
          {programs.map((prog) => (
            <button
              key={String(prog.id)}
              onClick={() => handleProgramChange(prog.id)}
              className={`px-4 py-2 text-[13px] font-medium transition-colors relative
                ${
                  activeProgram === prog.id
                    ? 'text-text-primary'
                    : 'text-text-tertiary hover:text-text-secondary'
                }`}
            >
              {prog.name}
              {activeProgram === prog.id && (
                <span className="absolute bottom-0 left-0 right-0 h-[2px] bg-accent-secondary rounded-full" />
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Two-card layout */}
      <div className="flex gap-4 items-start" style={{ minHeight: '440px' }}>
        {/* Left Card: Branches */}
        <div className="w-56 flex-shrink-0 rounded-[6px] border border-background-border/50 bg-background-secondary overflow-hidden">
          <div className="px-4 py-3 border-b border-background-border/40">
            <h2 className="text-[13px] font-semibold text-text-secondary uppercase tracking-wider">
              Branches
            </h2>
          </div>
          <div>
            {branches.map((dept) => {
              const hasHod = !!dept.current_hod;
              const isActive = activeBranch === dept.branch_id;
              return (
                <button
                  key={String(dept.branch_id)}
                  onClick={() => handleBranchChange(dept.branch_id)}
                  className={`w-full text-left px-4 py-3 flex items-center gap-2.5 text-sm transition-colors
                    ${
                      isActive
                        ? 'bg-accent-secondary/10 text-text-primary font-medium'
                        : 'text-text-secondary hover:bg-background-tertiary/60 hover:text-text-primary'
                    }`}
                >
                  <span
                    className={`w-[7px] h-[7px] rounded-full flex-shrink-0 ${hasHod ? 'bg-accent-success' : 'bg-amber-500'}`}
                  />
                  <span className="truncate">{dept.branch_name}</span>
                </button>
              );
            })}
          </div>
        </div>

        {/* Right Card: Faculty */}
        <div
          className="flex-1 rounded-[6px] border border-background-border/50 bg-background-secondary overflow-hidden flex flex-col"
          style={{ minHeight: '440px' }}
        >
          {activeDept ? (
            <>
              {/* Current HOD block — visually distinct */}
              <div className="px-5 py-4 bg-background-tertiary/50 border-b-2 border-accent-secondary/20">
                <div className="text-[11px] text-text-tertiary uppercase tracking-widest font-semibold mb-2">
                  Current HOD &mdash; {activeDept.branch_name}
                </div>
                {activeDept.current_hod ? (
                  <div className="flex items-center gap-3">
                    <span className="w-2 h-2 rounded-full bg-accent-success flex-shrink-0"></span>
                    <span className="text-base text-text-primary font-semibold">
                      {activeDept.current_hod.name}
                    </span>
                    <span className="text-xs text-text-tertiary">
                      {activeDept.current_hod.regdno}
                    </span>
                    <button
                      onClick={handleRemoveHOD}
                      disabled={actionLoading === activeBranch}
                      className="ml-auto px-2.5 py-1 text-xs text-red-400/80 hover:text-red-400 hover:bg-red-500/10 rounded-[4px] transition-colors disabled:opacity-50"
                    >
                      Remove
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-3">
                    <span className="w-2 h-2 rounded-full bg-amber-500 flex-shrink-0"></span>
                    <span className="text-sm text-amber-400/90">Not assigned</span>
                  </div>
                )}
              </div>

              {/* Faculty list header */}
              <div className="px-5 py-2.5 border-b border-background-border/30">
                <span className="text-[11px] text-text-tertiary uppercase tracking-widest font-semibold">
                  Available Faculty
                </span>
              </div>

              {/* Faculty list with inline assign */}
              <div className="flex-1 overflow-y-auto">
                {activeDept.available_faculty && activeDept.available_faculty.length > 0 ? (
                  <div>
                    {activeDept.available_faculty.map((f, idx) => {
                      const isSelected = selectedFaculty === f.regdno;
                      return (
                        <div
                          key={f.regdno}
                          className={`flex items-center gap-3 px-5 py-3 cursor-pointer transition-colors
                            ${idx < activeDept.available_faculty.length - 1 ? 'border-b border-background-border/20' : ''}
                            ${
                              isSelected
                                ? 'bg-accent-secondary/10'
                                : 'hover:bg-background-tertiary/40'
                            }`}
                          onClick={() => setSelectedFaculty(f.regdno)}
                        >
                          <input
                            type="radio"
                            name="faculty-select"
                            value={f.regdno}
                            checked={isSelected}
                            onChange={() => setSelectedFaculty(f.regdno)}
                            className="accent-accent-secondary w-4 h-4 flex-shrink-0"
                          />
                          <span className="text-sm text-text-primary">{f.name}</span>
                          <span className="text-text-tertiary text-xs">{f.regdno}</span>
                          {f.designation && (
                            <span className="text-text-muted text-xs">{f.designation}</span>
                          )}
                          {/* Inline assign button appears on selected row */}
                          {isSelected && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                handleAssignHOD();
                              }}
                              disabled={actionLoading === activeBranch}
                              className="ml-auto px-3 py-1 text-xs font-medium rounded-[4px] bg-accent-secondary/90 text-white
                                hover:bg-accent-secondary transition-colors
                                disabled:opacity-40 disabled:cursor-not-allowed whitespace-nowrap"
                            >
                              {actionLoading === activeBranch
                                ? 'Assigning...'
                                : activeDept.current_hod
                                  ? 'Reassign HOD'
                                  : 'Assign as HOD'}
                            </button>
                          )}
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <div className="px-5 py-10 text-sm text-text-tertiary text-center">
                    No faculty available in this branch.
                  </div>
                )}
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-text-tertiary text-[13px]">
              Select a branch to view faculty
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default HODManagement;
