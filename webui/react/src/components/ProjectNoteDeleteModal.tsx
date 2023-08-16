import React, { useCallback } from 'react';

import { Modal } from 'components/kit/Modal';
import { setProjectNotes } from 'services/api';
import { Project } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

interface Props {
  onClose?: () => void;
  project?: Project;
  pageNumber: number;
}

const ProjectNoteDeleteModalComponent: React.FC<Props> = ({
  onClose,
  pageNumber = 0,
  project,
}: Props) => {
  const handleSubmit = useCallback(async () => {
    if (!project?.id) return;
    try {
      await setProjectNotes({
        notes: project.notes.filter((note, idx) => idx !== pageNumber),
        projectId: project.id,
      });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete notes page.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [pageNumber, project?.id, project?.notes]);

  return (
    <Modal
      cancel
      danger
      size="small"
      submit={{
        handleError,
        handler: handleSubmit,
        text: 'Delete Page',
      }}
      title="Delete Page"
      onClose={onClose}>
      <p>
        Are you sure you want to delete&nbsp;
        <strong>&quot;{project?.notes?.[pageNumber]?.name ?? 'Untitled'}&quot;</strong>?
      </p>
      <p>This cannot be undone.</p>
    </Modal>
  );
};

export default ProjectNoteDeleteModalComponent;
