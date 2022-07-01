import { ExclamationCircleOutlined } from '@ant-design/icons';
import { ModalFuncProps } from 'antd';
import React, { useCallback, useEffect, useMemo } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Link from 'components/Link';
import useModal, { ModalCloseReason, ModalHooks } from 'hooks/useModal/useModal';
import { paths } from 'routes/utils';
import { formatDatetime } from 'shared/utils/datetime';
import { humanReadableBytes } from 'shared/utils/string';
import {
  CheckpointStorageType,
  CheckpointWorkloadExtended,
  ExperimentConfig,
  RunState,
} from 'types';
import { checkpointSize } from 'utils/workload';

import css from './useModalCheckpoint.module.scss';

export interface Props {
  checkpoint: CheckpointWorkloadExtended;
  config: ExperimentConfig;
  onClose?: (reason?: ModalCloseReason) => void;
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

const renderRow = (label: string, content: React.ReactNode): React.ReactNode => (
  <div className={css.row} key={label}>
    <div className={css.label}>{label}</div>
    <div className={css.content}>{content}</div>
  </div>
);

const renderResource = (resource: string, size: string): React.ReactNode => {
  return (
    <div className={css.resource} key={resource}>
      <div className={css.resourceName}>{resource}</div>
      <div className={css.resourceSpacer} />
      <div className={css.resourceSize}>{size}</div>
    </div>
  );
};

const useModalCheckpoint = ({
  checkpoint,
  config,
  title,
  onClose,
  ...props
}: Props): ModalHooks => {
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();

  const handleCancel = useCallback(() => onClose?.(ModalCloseReason.Cancel), [ onClose ]);

  const handleOk = useCallback(() => onClose?.(ModalCloseReason.Ok), [ onClose ]);

  const content = useMemo(() => {
    if (!checkpoint?.experimentId || !checkpoint?.resources) return null;

    const state = checkpoint.state as unknown as RunState;
    const totalSize = humanReadableBytes(checkpointSize(checkpoint));

    const searcherMetric = props.searcherValidation;
    const checkpointResources = checkpoint.resources;
    const resources = Object.keys(checkpoint.resources)
      .sort((a, b) => checkpointResources[a] - checkpointResources[b])
      .map(key => ({ name: key, size: humanReadableBytes(checkpointResources[key]) }));

    return (
      <>
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
              <HumanReadableNumber num={searcherMetric} />
              {`(${config.searcher.metric})`}
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
      </>
    );
  }, [ checkpoint, config, props.searcherValidation ]);

  const modalProps: ModalFuncProps = useMemo(() => ({
    content,
    icon: <ExclamationCircleOutlined />,
    okText: 'Register Checkpoint',
    onCancel: handleCancel,
    onOk: handleOk,
    title: title,
    width: 768,
  }), [ content, handleCancel, handleOk, title ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate ]);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(modalProps);
  }, [ modalProps, modalRef, openOrUpdate ]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalCheckpoint;
