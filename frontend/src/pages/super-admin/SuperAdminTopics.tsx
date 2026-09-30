import { useState, useEffect, FormEvent, ChangeEvent } from 'react';
import { superAdminSubjectAPI, superAdminTopicAPI } from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';
import {
  Tag,
  BookOpen,
  Plus,
  Edit2,
  Trash2,
  X,
  Check,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';

interface Topic {
  id: number;
  subject_id: number;
  name: string;
  description?: string;
  created_at?: string;
}

interface Subject {
  id: number;
  name: string;
  description?: string;
  topics?: Topic[];
  created_at?: string;
}

const SuperAdminTopics = () => {
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [expandedSubject, setExpandedSubject] = useState<number | null>(null);

  // Modals
  const [showCreateSubject, setShowCreateSubject] = useState<boolean>(false);
  const [showCreateTopic, setShowCreateTopic] = useState<boolean>(false);
  const [editingSubject, setEditingSubject] = useState<Subject | null>(null);
  const [editingTopic, setEditingTopic] = useState<Topic | null>(null);
  const [activeSubjectId, setActiveSubjectId] = useState<number | null>(null);

  const [subjectForm, setSubjectForm] = useState<{ name: string; description: string }>({
    name: '',
    description: '',
  });
  const [topicForm, setTopicForm] = useState<{ name: string; description: string }>({
    name: '',
    description: '',
  });

  useEffect(() => {
    fetchSubjects();
  }, []);

  const fetchSubjects = async () => {
    try {
      setLoading(true);
      const data = (await superAdminSubjectAPI.getAll(true)) as Subject[];
      setSubjects(Array.isArray(data) ? data : []);
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to load subjects');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateSubject = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      await superAdminSubjectAPI.create({
        name: subjectForm.name.trim(),
        description: subjectForm.description.trim(),
      });
      showSuccess('Subject created successfully!');
      setShowCreateSubject(false);
      setSubjectForm({ name: '', description: '' });
      fetchSubjects();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to create subject');
    }
  };

  const handleUpdateSubject = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!editingSubject) return;
    try {
      await superAdminSubjectAPI.update(editingSubject.id, {
        name: subjectForm.name.trim(),
        description: subjectForm.description.trim(),
      });
      showSuccess('Subject updated successfully!');
      setEditingSubject(null);
      setSubjectForm({ name: '', description: '' });
      fetchSubjects();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to update subject');
    }
  };

  const handleDeleteSubject = async (id: number) => {
    if (!confirm('Are you sure? This will delete the subject and all its topics.')) return;
    try {
      await superAdminSubjectAPI.delete(id);
      showSuccess('Subject deleted successfully!');
      fetchSubjects();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to delete subject');
    }
  };

  const handleCreateTopic = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!activeSubjectId) return;
    try {
      await superAdminTopicAPI.create({
        subject_id: activeSubjectId,
        name: topicForm.name.trim(),
        description: topicForm.description.trim(),
      });
      showSuccess('Topic created successfully!');
      setShowCreateTopic(false);
      setTopicForm({ name: '', description: '' });
      setActiveSubjectId(null);
      fetchSubjects();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to create topic');
    }
  };

  const handleUpdateTopic = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!editingTopic) return;
    try {
      await superAdminTopicAPI.update(editingTopic.id, {
        subject_id: editingTopic.subject_id,
        name: topicForm.name.trim(),
        description: topicForm.description.trim(),
      });
      showSuccess('Topic updated successfully!');
      setEditingTopic(null);
      setTopicForm({ name: '', description: '' });
      fetchSubjects();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to update topic');
    }
  };

  const handleDeleteTopic = async (id: number) => {
    if (
      !confirm('Are you sure you want to delete this topic? It may affect existing lab sessions.')
    )
      return;
    try {
      await superAdminTopicAPI.delete(id);
      showSuccess('Topic deleted successfully!');
      fetchSubjects();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to delete topic');
    }
  };

  const startEditSubject = (subject: Subject) => {
    setEditingSubject(subject);
    setSubjectForm({ name: subject.name, description: subject.description || '' });
  };

  const startEditTopic = (topic: Topic) => {
    setEditingTopic(topic);
    setTopicForm({ name: topic.name, description: topic.description || '' });
  };

  const openCreateTopic = (subjectId: number) => {
    setActiveSubjectId(subjectId);
    setTopicForm({ name: '', description: '' });
    setShowCreateTopic(true);
  };

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary mb-1">Subject & Topic Management</h1>
          <p className="text-text-secondary text-sm">
            Create subjects and manage topics under each subject for lab sessions and problems
          </p>
        </div>
        <button
          onClick={() => {
            setSubjectForm({ name: '', description: '' });
            setShowCreateSubject(true);
          }}
          className="btn btn-primary flex items-center gap-2"
        >
          <Plus className="w-4 h-4" />
          Add Subject
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-5">
        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-secondary">
          <BookOpen className="w-8 h-8 text-accent-secondary" />
          <div>
            <div className="text-3xl font-bold text-text-primary">
              {loading ? '...' : subjects.length}
            </div>
            <div className="text-sm text-text-secondary">Total Subjects</div>
          </div>
        </div>
        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-primary">
          <Tag className="w-8 h-8 text-accent-primary" />
          <div>
            <div className="text-3xl font-bold text-text-primary">
              {loading ? '...' : subjects.reduce((sum, s) => sum + (s.topics?.length || 0), 0)}
            </div>
            <div className="text-sm text-text-secondary">Total Topics</div>
          </div>
        </div>
      </div>

      {/* Subjects List */}
      {loading ? (
        <div className="text-center py-12 text-text-secondary">Loading subjects...</div>
      ) : subjects.length === 0 ? (
        <div className="text-center py-12">
          <BookOpen className="w-12 h-12 mx-auto text-text-muted mb-3" />
          <p className="text-text-secondary">No subjects yet</p>
          <button
            onClick={() => setShowCreateSubject(true)}
            className="mt-3 text-accent-secondary hover:underline text-sm"
          >
            Create your first subject
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {subjects.map((subject) => (
            <div key={subject.id} className="card-elevated">
              <div className="flex items-center justify-between">
                <div
                  className="flex items-center gap-3 min-w-0 flex-1 cursor-pointer"
                  onClick={() =>
                    setExpandedSubject(expandedSubject === subject.id ? null : subject.id)
                  }
                >
                  {expandedSubject === subject.id ? (
                    <ChevronDown className="w-5 h-5 text-text-muted shrink-0" />
                  ) : (
                    <ChevronRight className="w-5 h-5 text-text-muted shrink-0" />
                  )}
                  <div className="w-10 h-10 rounded-lg bg-accent-secondary/20 flex items-center justify-center shrink-0">
                    <BookOpen className="w-5 h-5 text-accent-secondary" />
                  </div>
                  <div className="min-w-0">
                    <div className="font-medium text-text-primary">{subject.name}</div>
                    {subject.description && (
                      <div className="text-sm text-text-secondary truncate">
                        {subject.description}
                      </div>
                    )}
                    <div className="text-xs text-text-muted">
                      {subject.topics?.length || 0} topics
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <button
                    onClick={() => openCreateTopic(subject.id)}
                    className="p-2 rounded-lg text-text-secondary hover:text-accent-secondary hover:bg-accent-secondary/10 transition-colors"
                    title="Add topic"
                  >
                    <Plus className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => startEditSubject(subject)}
                    className="p-2 rounded-lg text-text-secondary hover:text-text-primary hover:bg-background-tertiary transition-colors"
                    title="Edit"
                  >
                    <Edit2 className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => handleDeleteSubject(subject.id)}
                    className="p-2 rounded-lg text-text-secondary hover:text-red-400 hover:bg-red-500/10 transition-colors"
                    title="Delete"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {/* Topics under subject */}
              {expandedSubject === subject.id && (
                <div className="mt-4 pt-4 border-t border-background-border space-y-2">
                  {subject.topics && subject.topics.length > 0 ? (
                    subject.topics.map((topic) => (
                      <div
                        key={topic.id}
                        className="flex items-center justify-between bg-background-tertiary rounded-lg p-3"
                      >
                        <div className="flex items-center gap-3 min-w-0 flex-1">
                          <Tag className="w-4 h-4 text-accent-primary shrink-0" />
                          <div className="min-w-0">
                            <div className="text-sm font-medium text-text-primary">
                              {topic.name}
                            </div>
                            {topic.description && (
                              <div className="text-xs text-text-secondary truncate">
                                {topic.description}
                              </div>
                            )}
                          </div>
                        </div>
                        <div className="flex items-center gap-1 shrink-0">
                          <button
                            onClick={() => startEditTopic(topic)}
                            className="p-1.5 rounded-lg text-text-secondary hover:text-text-primary hover:bg-background-secondary transition-colors"
                            title="Edit"
                          >
                            <Edit2 className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => handleDeleteTopic(topic.id)}
                            className="p-1.5 rounded-lg text-text-secondary hover:text-red-400 hover:bg-red-500/10 transition-colors"
                            title="Delete"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>
                    ))
                  ) : (
                    <p className="text-text-muted text-sm py-2">
                      No topics under this subject yet.
                    </p>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Create Subject Modal */}
      {showCreateSubject && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-background-secondary border border-background-border rounded-xl p-6">
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-xl font-bold text-text-primary">Add New Subject</h2>
              <button
                onClick={() => setShowCreateSubject(false)}
                className="text-text-muted hover:text-text-primary"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreateSubject} className="space-y-4">
              <div>
                <label className="block text-xs text-text-secondary mb-1">Subject Name *</label>
                <input
                  type="text"
                  required
                  value={subjectForm.name}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setSubjectForm({ ...subjectForm, name: e.target.value })
                  }
                  placeholder="e.g. Data Structures, Operating Systems"
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                />
              </div>
              <div>
                <label className="block text-xs text-text-secondary mb-1">Description</label>
                <textarea
                  rows={3}
                  value={subjectForm.description}
                  onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                    setSubjectForm({ ...subjectForm, description: e.target.value })
                  }
                  placeholder="Optional description"
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary resize-none"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateSubject(false)}
                  className="btn btn-secondary"
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary flex items-center gap-2">
                  <Check className="w-4 h-4" /> Create Subject
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Subject Modal */}
      {editingSubject && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-background-secondary border border-background-border rounded-xl p-6">
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-xl font-bold text-text-primary">Edit Subject</h2>
              <button
                onClick={() => setEditingSubject(null)}
                className="text-text-muted hover:text-text-primary"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleUpdateSubject} className="space-y-4">
              <div>
                <label className="block text-xs text-text-secondary mb-1">Subject Name *</label>
                <input
                  type="text"
                  required
                  value={subjectForm.name}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setSubjectForm({ ...subjectForm, name: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                />
              </div>
              <div>
                <label className="block text-xs text-text-secondary mb-1">Description</label>
                <textarea
                  rows={3}
                  value={subjectForm.description}
                  onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                    setSubjectForm({ ...subjectForm, description: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary resize-none"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setEditingSubject(null)}
                  className="btn btn-secondary"
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary flex items-center gap-2">
                  <Check className="w-4 h-4" /> Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Create Topic Modal */}
      {showCreateTopic && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-background-secondary border border-background-border rounded-xl p-6">
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-xl font-bold text-text-primary">Add New Topic</h2>
              <button
                onClick={() => setShowCreateTopic(false)}
                className="text-text-muted hover:text-text-primary"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreateTopic} className="space-y-4">
              <div>
                <label className="block text-xs text-text-secondary mb-1">Subject</label>
                <input
                  type="text"
                  disabled
                  value={subjects.find((s) => s.id === activeSubjectId)?.name || ''}
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm opacity-60"
                />
              </div>
              <div>
                <label className="block text-xs text-text-secondary mb-1">Topic Name *</label>
                <input
                  type="text"
                  required
                  value={topicForm.name}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setTopicForm({ ...topicForm, name: e.target.value })
                  }
                  placeholder="e.g. Arrays, Dynamic Programming"
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                />
              </div>
              <div>
                <label className="block text-xs text-text-secondary mb-1">Description</label>
                <textarea
                  rows={3}
                  value={topicForm.description}
                  onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                    setTopicForm({ ...topicForm, description: e.target.value })
                  }
                  placeholder="Optional description"
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary resize-none"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateTopic(false)}
                  className="btn btn-secondary"
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary flex items-center gap-2">
                  <Check className="w-4 h-4" /> Create Topic
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Topic Modal */}
      {editingTopic && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-background-secondary border border-background-border rounded-xl p-6">
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-xl font-bold text-text-primary">Edit Topic</h2>
              <button
                onClick={() => setEditingTopic(null)}
                className="text-text-muted hover:text-text-primary"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleUpdateTopic} className="space-y-4">
              <div>
                <label className="block text-xs text-text-secondary mb-1">Topic Name *</label>
                <input
                  type="text"
                  required
                  value={topicForm.name}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setTopicForm({ ...topicForm, name: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                />
              </div>
              <div>
                <label className="block text-xs text-text-secondary mb-1">Description</label>
                <textarea
                  rows={3}
                  value={topicForm.description}
                  onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                    setTopicForm({ ...topicForm, description: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary resize-none"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setEditingTopic(null)}
                  className="btn btn-secondary"
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary flex items-center gap-2">
                  <Check className="w-4 h-4" /> Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default SuperAdminTopics;
