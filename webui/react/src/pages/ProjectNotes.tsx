import { useModal } from 'determined-ui/Modal';
import Notes from 'determined-ui/Notes';
import React, { useCallback, useRef, useState } from 'react';
import { unstable_useBlocker } from 'react-router-dom';

import { useSetDynamicTabBar } from 'components/DynamicTabs';
import ProjectNoteDeleteModalComponent from 'components/ProjectNoteDeleteModal';
import usePermissions from 'hooks/usePermissions';
import { addProjectNote, setProjectNotes } from 'services/api';
import { Note, Project } from 'types';
import handleError from 'utils/error';

interface Props {
  fetchProject: () => void;
  project: Project;
}

const ProjectNotes: React.FC<Props> = ({ project, fetchProject }) => {
  const containerRef = useRef(null);
  const [pageNumber, setPageNumber] = useState<number>(0);
  const handleNewNotesPage = useCallback(async () => {
    if (!project?.id) return;
    try {
      await addProjectNote({ contents: '', id: project.id, name: 'Untitled' });
      await fetchProject();
    } catch (e) {
      handleError(containerRef, e);
    }
  }, [fetchProject, project?.id]);

  const { canCreateExperiment } = usePermissions();
  const editPermission = canCreateExperiment({ workspace: { id: project.workspaceId } });

  const handleSaveNotes = useCallback(
    async (notes: Note[]) => {
      if (!project?.id) return;
      try {
        await setProjectNotes({ notes, projectId: project.id });
        await fetchProject();
      } catch (e) {
        handleError(containerRef, e);
      }
    },
    [fetchProject, project?.id],
  );

  const ProjectNoteDeleteModal = useModal(ProjectNoteDeleteModalComponent);

  const handleDeleteNote = useCallback(
    (pageNumber: number) => {
      if (!project?.id) return;
      try {
        setPageNumber(pageNumber);
        ProjectNoteDeleteModal.open();
      } catch (e) {
        handleError(containerRef, e);
      }
    },
    [ProjectNoteDeleteModal, project?.id],
  );

  useSetDynamicTabBar(<></>);

  return (
    <div ref={containerRef}>
      <Notes
        disabled={project?.archived || !editPermission}
        multiple
        notes={project?.notes ?? []}
        onDelete={handleDeleteNote}
        onError={handleError}
        onNewPage={handleNewNotesPage}
        onPageUnloadHook={unstable_useBlocker}
        onSave={handleSaveNotes}
      />
      <ProjectNoteDeleteModal.Component
        pageNumber={pageNumber}
        project={project}
        onClose={fetchProject}
      />
    </div>
  );
};

export default ProjectNotes;
