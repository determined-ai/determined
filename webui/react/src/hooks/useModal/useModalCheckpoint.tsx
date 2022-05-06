import { ExclamationCircleOutlined } from '@ant-design/icons';
import { ModalFuncProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Link from 'components/Link';
import useCreateModelModal from 'hooks/useCreateModelModal';
import useRegisterCheckpointModal from 'hooks/useRegisterCheckpointModal';
import { paths } from 'routes/utils';
import {
  CheckpointStorageType,
  CheckpointWorkloadExtended,
  ExperimentConfig,
  RunState,
} from 'types';
import { formatDatetime } from 'utils/datetime';
import { humanReadableBytes } from 'utils/string';
import { checkpointSize } from 'utils/workload';

import useModal, { ModalHooks } from './useModal';
import css from './useModalCheckpoint.module.scss';

interface Props {
  checkpoint: CheckpointWorkloadExtended;
  config: ExperimentConfig;
  searcherValidation?: number;
  title: string;
}

const getStorageLocation = (
  config: ExperimentConfig,
  checkpoint: CheckpointWorkloadExtended,
): string => {
  const hostPath = config.checkpointStorage?.hostPath;
  const storagePath = config.checkpointStorage?.storagePath;
  let location = '';
  switch (config.checkpointStorage?.type) {
    case CheckpointStorageType.AWS:
      location = `s3://${config.checkpointStorage.bucket || ''}`;
      break;
    case CheckpointStorageType.GCS:
      location = `gs://${config.checkpointStorage.bucket || ''}`;
      break;
    case CheckpointStorageType.SharedFS:
      if (hostPath && storagePath) {
        location = storagePath.startsWith('/') ?
          `file://${storagePath}` : `file://${hostPath}/${storagePath}`;
      } else if (hostPath) {
        location = `file://${hostPath}`;
      }
      break;
  }
  return `${location}/${checkpoint.uuid}`;
};

const renderRow = (label: string, content: React.ReactNode): React.ReactNode => {
  return (
    <div className={css.row} key={label}>
      <div className={css.label}>{label}</div>
      <div className={css.content}>{content}</div>
    </div>
  );
};

const renderResource = (resource: string, size: string): React.ReactNode => {
  return (
    <div className={css.resource} key={resource}>
      <div className={css.resourceName}>{resource}</div>
      <div className={css.resourceSpacer} />
      <div className={css.resourceSize}>{size}</div>
    </div>
  );
};

const useModalCheckpoint = ({ checkpoint, config, title, ...props }: Props): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();

  const { showModal: showCreateModelModal } = useCreateModelModal();

  const handleRegisterCheckpointClose = useCallback((checkpointUuid?: string) => {
    if (checkpointUuid) showCreateModelModal({ checkpointUuid });
  }, [ showCreateModelModal ]);

  const { showModal: showRegisterCheckpointModal } = useRegisterCheckpointModal(
    handleRegisterCheckpointClose,
  );

  const launchRegisterCheckpointModal = useCallback(() => {
    if (!checkpoint.uuid) return;
    showRegisterCheckpointModal({ checkpointUuid: checkpoint.uuid });
  }, [ checkpoint.uuid, showRegisterCheckpointModal ]);

  const handleOk = useCallback(() => {
    launchRegisterCheckpointModal();
  }, [ launchRegisterCheckpointModal ]);

  const getContent = useCallback(() => {
    if(!checkpoint?.experimentId || !checkpoint?.resources){
      return null;
    }

    const state = checkpoint.state as unknown as RunState;
    const totalSize = humanReadableBytes(checkpointSize(checkpoint));

    const searcherMetric = props.searcherValidation;
    const checkpointResources = checkpoint.resources;
    const resources = Object.keys(checkpoint.resources)
      .sort((a, b) => checkpointResources[a] - checkpointResources[b])
      .map(key => ({ name: key, size: humanReadableBytes(checkpointResources[key]) }));
    return (
      <div className={css.base}>
        {renderRow(
          'Source', (
            <div className={css.source}>
              <Link path={paths.experimentDetails(checkpoint.experimentId)}>
                Experiment {checkpoint.experimentId}
              </Link>
              <span className={css.sourceDivider} />
              <Link path={paths.trialDetails(checkpoint.trialId, checkpoint.experimentId)}>
                Trial {checkpoint.trialId}
              </Link>
              <span className={css.sourceDivider} />
              <span>Batch {checkpoint.totalBatches}</span>
            </div>
          ),
        )}
        {renderRow('State', <Badge state={state} type={BadgeType.State} />)}
        {checkpoint.uuid && renderRow('UUID', checkpoint.uuid)}
        {renderRow('Location', getStorageLocation(config, checkpoint))}
        {searcherMetric && renderRow(
          'Validation Metric',
          <>
            <HumanReadableNumber num={searcherMetric} /> {`(${config.searcher.metric})`}
          </>,
        )}
        {checkpoint.endTime && renderRow('End Time', formatDatetime(checkpoint.endTime))}
        {renderRow('Total Size', totalSize)}
        {resources.length !== 0 && renderRow(
          'Resources', (
            <div className={css.resources}>
              {resources.map(resource => renderResource(resource.name, resource.size))}
            </div>
          ),
        )}
      </div>
    );
  }, [ checkpoint, config, props.searcherValidation ]);

  const modalProps: ModalFuncProps = useMemo(() => {
    return {
      content: getContent(),
      icon: <ExclamationCircleOutlined />,
      okText: 'Register Checkpoint',
      onOk: handleOk,
      title: title,
      width: 768,
    };
  }, [ handleOk, getContent, title ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalCheckpoint;
