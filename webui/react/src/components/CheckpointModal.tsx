import { Modal } from 'antd';
import React, { useMemo } from 'react';

import HumanReadableFloat from 'components/HumanReadableFloat';
import { paths } from 'routes/utils';
import { CheckpointDetail, CheckpointStorageType, CheckpointWorkload, CheckpointWorkloadExtended,
  ExperimentConfig } from 'types';
import { formatDatetime } from 'utils/date';
import { humanReadableBytes } from 'utils/string';
import { checkpointSize, getBatchNumber } from 'utils/types';

import css from './CheckpointModal.module.scss';
import Link from './Link';

interface Props {
  checkpoint: CheckpointWorkloadExtended | CheckpointDetail;
  config: ExperimentConfig;
  onHide?: () => void;
  searcherValidation?: number;
  show?: boolean;
  title: string;
}

const getStorageLocation = (
  config: ExperimentConfig,
  checkpoint: CheckpointDetail | CheckpointWorkload,
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

const CheckpointModal: React.FC<Props> = (
  { config, checkpoint, onHide, show, title, ...props }: Props,
) => {
  const totalSize = useMemo(() => {
    return humanReadableBytes(checkpointSize(checkpoint));
  }, [ checkpoint ]);

  const resources = useMemo(() => {
    if (checkpoint.resources === undefined) return [];
    const checkpointResources = checkpoint.resources;
    return Object.keys(checkpoint.resources)
      .sort((a, b) => checkpointResources[a] - checkpointResources[b])
      .map(key => ({ name: key, size: humanReadableBytes(checkpointResources[key]) }));
  }, [ checkpoint.resources ]);

  const totalBatchesProcessed = getBatchNumber(checkpoint);

  const searcherMetric = props.searcherValidation !== undefined ?
    props.searcherValidation :
    ('validationMetric' in checkpoint ? checkpoint.validationMetric : undefined);

  if (!checkpoint.experimentId || !checkpoint.trialId) {
    return null;
  }

  return (
    <Modal
      footer={null}
      title={title}
      visible={show}
      width={768}
      onCancel={onHide}>
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
              <span>Batch {totalBatchesProcessed}</span>
            </div>
          ),
        )}
        {checkpoint.uuid && renderRow('UUID', checkpoint.uuid)}
        {renderRow('Location', getStorageLocation(config, checkpoint))}
        {searcherMetric && renderRow(
          'Validation Metric',
          <>
            <HumanReadableFloat num={searcherMetric} /> {`(${config.searcher.metric})`}
          </>,
        )}
        {renderRow('Start Time', formatDatetime(checkpoint.startTime))}
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
    </Modal>
  );
};

export default CheckpointModal;
