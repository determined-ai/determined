import { Modal } from 'hew/Modal';
import { ShirtSize } from 'hew/Theme';
import React from 'react';

import Grid from 'components/Grid';
import Link from 'components/Link';

import css from './TaskListModalComponent.module.scss';

export interface TensorBoardSource {
  id: number;
  path: string;
  type: string;
}

export interface SourceInfo {
  path: string;
  plural: string;
  sources: TensorBoardSource[];
}

interface Props {
  title: string;
  sourcesModal?: SourceInfo;
  onClose: () => void;
}

const TaskListModalComponent: React.FC<Props> = ({ onClose, sourcesModal, title }: Props) => {
  return (
    <Modal
      size="medium"
      submit={{
        handleError: () => {},
        handler: onClose,
        text: 'Close',
      }}
      title={title}
      onClose={onClose}>
      <div className={css.sourceLinks}>
        <Grid gap={ShirtSize.Medium} minItemWidth={120}>
          {sourcesModal?.sources.map((source) => (
            <Link key={source.id} path={source.path}>
              {source.type} {source.id}
            </Link>
          ))}
        </Grid>
      </div>
    </Modal>
  );
};

export default TaskListModalComponent;
