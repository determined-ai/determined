import React, { useCallback, useMemo } from 'react';

import { Modal } from 'components/kit/Modal';
import { ModalCloseReason } from 'hooks/useModal/useModal';
import { paths } from 'routes/utils';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import {
  CheckpointState,
  CheckpointStorageType,
  CheckpointWorkloadExtended,
  CoreApiGenericCheckpoint,
  ExperimentConfig,
} from 'types';
import { formatDatetime } from 'utils/datetime';
import handleError from 'utils/error';
import { humanReadableBytes } from 'utils/string';
import { checkpointSize } from 'utils/workload';

import css from './CheckpointModal.module.scss';
import HumanReadableNumber from './HumanReadableNumber';
import Button from './kit/Button';
import useConfirm from './kit/useConfirm';
import Link from './Link';
import { StateBadge } from './StateBadge';

export interface Props {
  checkpoint?: CheckpointWorkloadExtended | CoreApiGenericCheckpoint;
  children?: React.ReactNode;
  config: ExperimentConfig;
  onClose: (reason?: ModalCloseReason) => void;
  searcherValidation?: number;
  title: string;
}

const getStorageLocation = (
  config: ExperimentConfig,
  checkpoint: CheckpointWorkloadExtended | CoreApiGenericCheckpoint,
): string => {
  const hostPath = config.checkpointStorage?.hostPath;
  const storagePath = config.checkpointStorage?.storagePath;
  let location = '';
  switch (config.checkpointStorage?.type) {
    case CheckpointStorageType.AWS:
    case CheckpointStorageType.S3:
      location = `s3://${config.checkpointStorage.bucket || ''}`;
      break;
    case CheckpointStorageType.GCS:
      location = `gs://${config.checkpointStorage.bucket || ''}`;
      break;
    case CheckpointStorageType.SharedFS:
      if (hostPath && storagePath) {
        location = storagePath.startsWith('/')
          ? `file://${storagePath}`
          : `file://${hostPath}/${storagePath}`;
      } else if (hostPath) {
        location = `file://${hostPath}`;
      }
      break;
    case CheckpointStorageType.AZURE:
      // type from api doesn't have azure-specific props
      break;
    case undefined:
      // shouldn't happen?
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

const CheckpointModalComponent: React.FC<Props> = ({
  checkpoint,
  config,
  title,
  onClose,
  ...props
}: Props) => {
  const confirm = useConfirm();

  const handleCancel = useCallback(() => onClose(ModalCloseReason.Cancel), [onClose]);

  const handleOk = useCallback(() => onClose(ModalCloseReason.Ok), [onClose]);

  const handleDelete = useCallback(async () => {
    if (!checkpoint?.uuid) return;
    await readStream(detApi.Checkpoint.deleteCheckpoints({ checkpointUuids: [checkpoint.uuid] }));
  }, [checkpoint]);

  const onClickDelete = useCallback(() => {
    const content = `Are you sure you want to request checkpoint deletion for batch
${checkpoint?.totalBatches}. This action may complete or fail without further notification.`;

    confirm({
      content,
      danger: true,
      okText: 'Request Delete',
      onConfirm: handleDelete,
      onError: handleError,
      title: 'Confirm Checkpoint Deletion',
    });
  }, [checkpoint?.totalBatches, confirm, handleDelete]);

  const content = useMemo(() => {
    if (!checkpoint?.experimentId || !checkpoint?.resources) return null;

    const state = checkpoint.state;
    const totalSize = humanReadableBytes(checkpointSize(checkpoint));

    const searcherMetric = props.searcherValidation;
    const checkpointResources = checkpoint.resources;
    const resources = Object.keys(checkpoint.resources)
      .sort((a, b) => checkpointResources[a] - checkpointResources[b])
      .map((key) => ({ name: key, size: humanReadableBytes(checkpointResources[key]) }));

    return (
      <div className={css.base}>
        {renderRow(
          'Source',
          <div className={css.source}>
            <Link path={paths.experimentDetails(checkpoint.experimentId)}>
              Experiment {checkpoint.experimentId}
            </Link>
            {checkpoint.trialId && (
              <>
                <span className={css.sourceDivider} />
                <Link path={paths.trialDetails(checkpoint.trialId, checkpoint.experimentId)}>
                  Trial {checkpoint.trialId}
                </Link>
              </>
            )}
            <span className={css.sourceDivider} />
            <span>Batch {checkpoint.totalBatches}</span>
          </div>,
        )}
        {renderRow('State', <StateBadge state={state} />)}
        {checkpoint.uuid && renderRow('UUID', checkpoint.uuid)}
        {renderRow('Location', getStorageLocation(config, checkpoint))}
        {searcherMetric &&
          renderRow(
            'Validation Metric',
            <>
              <HumanReadableNumber num={searcherMetric} />
              {`(${config.searcher.metric})`}
            </>,
          )}
        {'endTime' in checkpoint &&
          checkpoint?.endTime &&
          renderRow('Ended', formatDatetime(checkpoint.endTime))}
        {renderRow(
          'Total Size',
          <div className={css.size}>
            <span>{totalSize}</span>
            {checkpoint.uuid && state !== CheckpointState.Deleted && (
              <Button danger type="text" onClick={onClickDelete}>
                {'Request Checkpoint Deletion'}
              </Button>
            )}
          </div>,
        )}
        {resources.length !== 0 &&
          renderRow(
            'Resources',
            <div className={css.resources}>
              {resources.map((resource) => renderResource(resource.name, resource.size))}
            </div>,
          )}
      </div>
    );
  }, [checkpoint, config, props.searcherValidation, onClickDelete]);

  return (
    <Modal
      cancel
      submit={{
        handleError,
        handler: handleOk,
        text: 'Register Checkpoint',
      }}
      title={title}
      onClose={handleCancel}>
      {content}
    </Modal>
  );
};

export default CheckpointModalComponent;
