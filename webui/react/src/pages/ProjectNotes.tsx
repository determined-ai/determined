import { useModal } from 'hew/Modal';
import Notes from 'hew/RichTextEditor';
import React, { useCallback, useState } from 'react';
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
  const [pageNumber, setPageNumber] = useState<number>(0);
  const handleNewNotesPage = useCallback(async () => {
    if (!project?.id) return;
    try {
      await addProjectNote({ contents: '', id: project.id, name: 'Untitled' });
      await fetchProject();
    } catch (e) {
      handleError(e);
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
        handleError(e);
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
        handleError(e);
      }
    },
    [ProjectNoteDeleteModal, project?.id],
  );

  useSetDynamicTabBar(<></>);

  return (
    <>
      <Notes
        disabled={project?.archived || !editPermission}
        docs={project?.notes ?? []}
        multiple
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
    </>
  );
};

export default ProjectNotes;
