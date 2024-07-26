import Breadcrumb from 'hew/Breadcrumb';
import Button from 'hew/Button';
import Glossary, { InfoRow } from 'hew/Glossary';
import { Modal, ModalCloseReason } from 'hew/Modal';
import useConfirm from 'hew/useConfirm';
import React, { useCallback, useMemo } from 'react';

import useFeature from 'hooks/useFeature';
import { paths } from 'routes/utils';
import { deleteCheckpoints } from 'services/api';
import {
  CheckpointState,
  CheckpointStorageType,
  CheckpointWorkloadExtended,
  CoreApiGenericCheckpoint,
  ExperimentConfig,
} from 'types';
import { formatDatetime } from 'utils/datetime';
import handleError, { DetError, ErrorType } from 'utils/error';
import { createPachydermLineageLink } from 'utils/integrations';
import { isAbsolutePath } from 'utils/routes';
import { humanReadableBytes } from 'utils/string';
import { checkpointSize } from 'utils/workload';

import Badge, { BadgeType } from './Badge';
import css from './CheckpointModal.module.scss';
import HumanReadableNumber from './HumanReadableNumber';
import Link from './Link';

export interface Props {
  checkpoint?: CheckpointWorkloadExtended | CoreApiGenericCheckpoint;
  children?: React.ReactNode;
  config: ExperimentConfig;
  onClose?: (reason?: ModalCloseReason) => void;
  searcherValidation?: number;
  title: string;
}

const getStorageLocation = (
  config: ExperimentConfig,
  checkpoint: CheckpointWorkloadExtended | CoreApiGenericCheckpoint,
): string => {
  const hostPath = config.checkpointStorage?.hostPath;
  const storagePath = config.checkpointStorage?.storagePath;
  const containerPath = config.checkpointStorage?.containerPath;
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
          ? `file:/${storagePath}`
          : `file://${hostPath}/${storagePath}`;
      } else if (hostPath) {
        location = `file:${isAbsolutePath(hostPath.replace(' ', '')) ? '/' : '//'}${hostPath}`;
      }
      break;
    case CheckpointStorageType.DIRECTORY:
      if (containerPath) {
        location = `file:${
          isAbsolutePath(containerPath.replace(' ', '')) ? '/' : '//'
        }${containerPath}`;
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

const CheckpointModalComponent: React.FC<Props> = ({
  checkpoint,
  config,
  title,
  onClose,
  ...props
}: Props) => {
  const confirm = useConfirm();

  const handleCancel = useCallback(() => onClose?.('Cancel'), [onClose]);

  const handleOk = useCallback(() => onClose?.('Ok'), [onClose]);
  const f_flat_runs = useFeature().isOn('flat_runs');

  const handleDelete = useCallback(async () => {
    if (!checkpoint?.uuid) return;
    try {
      await deleteCheckpoints({ checkpointUuids: [checkpoint.uuid] });
    } catch (e) {
      if (e instanceof DetError && e.type === ErrorType.Server) {
        e.silent = false;
      }
      // modal error handling overwrites error message
      handleError(e);
    }
  }, [checkpoint]);

  const onClickDelete = useCallback(() => {
    const content = `Are you sure you want to request checkpoint deletion for batch
${checkpoint?.totalBatches}? This action may complete or fail without further notification.`;

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
    if (!checkpoint?.experimentId || !checkpoint?.resources) return;

    const state = checkpoint.state;
    const totalSize = humanReadableBytes(checkpointSize(checkpoint));

    const searcherMetric = props.searcherValidation;
    const checkpointResources = checkpoint.resources;
    const resources = Object.keys(checkpoint.resources)
      .sort((a, b) => checkpointResources[a] - checkpointResources[b])
      .map((key) => ({ name: key, size: humanReadableBytes(checkpointResources[key]) }));

    const glossaryContent: InfoRow[] = [
      {
        label: 'Source',
        value: (
          <Breadcrumb>
            <Breadcrumb.Item>
              <Link path={paths.experimentDetails(checkpoint.experimentId)}>
                {f_flat_runs ? 'Search' : 'Experiment'} {checkpoint.experimentId}
              </Link>
            </Breadcrumb.Item>
            {checkpoint.trialId && (
              <Breadcrumb.Item>
                <Link path={paths.trialDetails(checkpoint.trialId, checkpoint.experimentId)}>
                  {f_flat_runs ? 'Run' : 'Trial'} {checkpoint.trialId}
                </Link>
              </Breadcrumb.Item>
            )}
            <Breadcrumb.Item>Batch {checkpoint.totalBatches}</Breadcrumb.Item>
          </Breadcrumb>
        ),
      },
      { label: 'State', value: <Badge state={state} type={BadgeType.State} /> },
    ];

    if (config.integrations?.pachyderm !== undefined) {
      const pachydermData = config.integrations.pachyderm;
      const url = createPachydermLineageLink(pachydermData);

      glossaryContent.splice(1, 0, {
        label: 'Data Input',
        value: <Link path={url}>{pachydermData.dataset.repo}</Link>,
      });
    }

    if (checkpoint.uuid) glossaryContent.push({ label: 'UUID', value: checkpoint.uuid });
    glossaryContent.push({ label: 'Location', value: getStorageLocation(config, checkpoint) });
    if (searcherMetric)
      glossaryContent.push({
        label: 'Validation Metric',
        value: (
          <>
            <HumanReadableNumber num={searcherMetric} />
            (config.searcher.metric)
          </>
        ),
      });
    if ('endTime' in checkpoint && checkpoint?.endTime)
      glossaryContent.push({ label: 'Ended', value: formatDatetime(checkpoint.endTime) });
    glossaryContent.push({
      label: 'Total Size',
      value: (
        <div className={css.size}>
          <span>{totalSize}</span>
          {checkpoint.uuid && state !== CheckpointState.Deleted && (
            <Button danger type="text" onClick={onClickDelete}>
              {'Request Checkpoint Deletion'}
            </Button>
          )}
        </div>
      ),
    });
    if (resources.length > 0)
      glossaryContent.push({
        label: 'Resources',
        value: (
          <Glossary content={resources.map(({ name, size }) => ({ label: name, value: size }))} />
        ),
      });
    return glossaryContent;
  }, [checkpoint, config, f_flat_runs, props.searcherValidation, onClickDelete]);

  return (
    <Modal
      cancel
      submit={{
        disabled: checkpoint?.state === CheckpointState.Deleted,
        handleError,
        handler: handleOk,
        text: 'Register Checkpoint',
      }}
      title={title}
      onClose={handleCancel}>
      <div className={css.base}>
        <Glossary content={content} />
      </div>
    </Modal>
  );
};

export default CheckpointModalComponent;
