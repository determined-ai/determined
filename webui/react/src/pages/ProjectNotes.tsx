import React, { useCallback, useMemo } from 'react';

import { useSetDynamicTabBar } from 'components/DynamicTabs';
import Button from 'components/kit/Button';
import PaginatedNotesCard from 'components/PaginatedNotesCard';
import useModalProjectNoteDelete from 'hooks/useModal/Project/useModalProjectNoteDelete';
import { addProjectNote, setProjectNotes } from 'services/api';
import { Note, Project } from 'types';
import handleError from 'utils/error';

import css from './ProjectDetails.module.scss';

interface Props {
  fetchProject: () => void;
  project: Project;
}

const ProjectNotes: React.FC<Props> = ({ project, fetchProject }) => {
  const handleNewNotesPage = useCallback(async () => {
    if (!project?.id) return;
    try {
      await addProjectNote({ contents: '', id: project.id, name: 'Untitled' });
      await fetchProject();
    } catch (e) {
      handleError(e);
    }
  }, [fetchProject, project?.id]);

  const handleSaveNotes = useCallback(
    async (notes: Note[]) => {
      if (!project?.id) return;
      try {
        await setProjectNotes({ notes, projectId: project.id });
        await fetchProject();
      } catch (e) {
        handleError(e);
      }
    },
    [fetchProject, project?.id],
  );

  const { contextHolder: modalProjectNodeDeleteContextHolder, modalOpen: openNoteDelete } =
    useModalProjectNoteDelete({ onClose: fetchProject, project });

  const handleDeleteNote = useCallback(
    (pageNumber: number) => {
      if (!project?.id) return;
      try {
        openNoteDelete({ pageNumber });
      } catch (e) {
        handleError(e);
      }
    },
    [openNoteDelete, project?.id],
  );

  const notesTabBarContent = useMemo(
    () => (
      <div className={css.tabOptions}>
        <Button type="text" onClick={handleNewNotesPage}>
          + New Page
        </Button>
      </div>
    ),
    [handleNewNotesPage],
  );

  useSetDynamicTabBar(notesTabBarContent);

  return (
    <>
      <PaginatedNotesCard
        disabled={project?.archived}
        notes={project?.notes ?? []}
        onDelete={handleDeleteNote}
        onNewPage={handleNewNotesPage}
        onSave={handleSaveNotes}
      />
      {modalProjectNodeDeleteContextHolder}
    </>
  );
};

export default ProjectNotes;
