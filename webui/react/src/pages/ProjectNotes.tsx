import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Dropdown, Menu, Modal, Space } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Link from 'components/Link';
import Page from 'components/Page';
import PaginatedNotesCard from 'components/PaginatedNotesCard';
import { useStore } from 'contexts/Store';
import useModalProjectNoteDelete from 'hooks/useModal/Project/useModalProjectNoteDelete';
import {
  activateExperiment, addProjectNote, archiveExperiment, cancelExperiment, deleteExperiment,
  getExperimentLabels, getExperiments, getProject, killExperiment, openOrCreateTensorBoard,
  patchExperiment, pauseExperiment, setProjectNotes, unarchiveExperiment,
} from 'services/api';
import {

  Note,
  Project,

} from 'types';
import handleError from 'utils/error';

import css from './ProjectDetails.module.scss';

interface Props {
  fetchProject: () => void;
  project: Project
}

const ProjectNotes: React.FC<Props> = ({ project, fetchProject }) => {
  const { users, auth: { user } } = useStore();

  // const [ project, setProject ] = useState<Project>();

  const handleNewNotesPage = useCallback(async () => {
    if (!project?.id) return;
    try {
      await addProjectNote({ contents: '', id: project.id, name: 'Untitled' });
      await fetchProject();
    } catch (e) { handleError(e); }
  }, [ fetchProject, project?.id ]);

  const handleSaveNotes = useCallback(async (notes: Note[]) => {
    if (!project?.id) return;
    try {
      await setProjectNotes({ notes, projectId: project.id });
      await fetchProject();
    } catch (e) { handleError(e); }
  }, [ fetchProject, project?.id ]);

  const {
    contextHolder: modalProjectNodeDeleteContextHolder,
    modalOpen: openNoteDelete,
  } = useModalProjectNoteDelete({ onClose: fetchProject, project });

  const handleDeleteNote = useCallback((pageNumber: number) => {
    if (!project?.id) return;
    try {
      openNoteDelete({ pageNumber });
    } catch (e) { handleError(e); }
  }, [ openNoteDelete, project?.id ]);

  const tabBarExtraContent = (
    <div className={css.tabOptions}>
      <Button type="text" onClick={handleNewNotesPage}>+ New Page</Button>
    </div>
  );

  return (
    <PaginatedNotesCard
      disabled={project?.archived}
      notes={project?.notes ?? []}
      onDelete={handleDeleteNote}
      onNewPage={handleNewNotesPage}
      onSave={handleSaveNotes}
    />
  );

};
