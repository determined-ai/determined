import { Modal } from 'antd';
import React, { useMemo } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import { CheckpointDetail, CheckpointStorageType, ExperimentConfig, RunState } from 'types';
import { formatDatetime } from 'utils/date';
import { humanReadableBytes, humanReadableFloat } from 'utils/string';

import css from './CheckpointModal.module.scss';
import Link from './Link';

interface Props {
  checkpoint: CheckpointDetail;
  config: ExperimentConfig;
  onHide?: () => void;
  show?: boolean;
  title: string;
}

const getStorageLocation = (config: ExperimentConfig, checkpoint: CheckpointDetail): string => {
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

const CheckpointModal: React.FC<Props> = ({ config, checkpoint, onHide, show, title }: Props) => {
  const state = checkpoint.state as unknown as RunState;

  const totalSize = useMemo(() => {
    const total = Object.values(checkpoint.resources).reduce((acc, size) => acc + size, 0);
    return humanReadableBytes(total);
  }, [ checkpoint.resources ]);

  const resources = useMemo(() => {
    return Object.keys(checkpoint.resources)
      .sort((a, b) => checkpoint.resources[a] - checkpoint.resources[b])
      .map(key => ({ name: key, size: humanReadableBytes(checkpoint.resources[key]) }));
  }, [ checkpoint.resources ]);

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
              <Link path={`/ui/experiments/${checkpoint.experimentId}`}>
              Experiment {checkpoint.experimentId}
              </Link>
              <span className={css.sourceDivider} />
              <Link path={`/ui/trials/${checkpoint.trialId}`}>Trial {checkpoint.trialId}</Link>
              <span className={css.sourceDivider} />
              <span>Batch {checkpoint.batch}</span>
            </div>
          ),
        )}
        {renderRow('State', <Badge state={state} type={BadgeType.State} />)}
        {checkpoint.uuid && renderRow('UUID', checkpoint.uuid)}
        {renderRow('Location', getStorageLocation(config, checkpoint))}
        {checkpoint.validationMetric && renderRow(
          'Validation Metric',
          `${humanReadableFloat(checkpoint.validationMetric)} (${config.searcher.metric})`,
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
