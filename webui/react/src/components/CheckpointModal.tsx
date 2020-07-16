import { Modal } from 'antd';
import React, { useCallback } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import { CheckpointDetail, CheckpointStorageType, ExperimentConfig } from 'types';
import { formatDatetime } from 'utils/date';
import { humanReadableFloat } from 'utils/string';

import css from './CheckpointModal.module.scss';

interface Props {
  checkpoint: CheckpointDetail;
  config: ExperimentConfig;
  onHide?: () => void;
  show?: boolean;
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
      location = `gs://${config.checkpointStorage.hostPath || ''}`;
      break;
    case CheckpointStorageType.SharedFS:
      if (hostPath && storagePath) {
        location = storagePath.startsWith('/') ?
          `file://${hostPath}/${storagePath}` :
          `file://${storagePath}`;
      } else if (hostPath) {
        location = `file://${hostPath}`;
      }
      break;
  }
  return `${location}/${checkpoint.id}`;
};

const CheckpointModal: React.FC<Props> = ({ config, checkpoint, onHide, show }: Props) => {
  const handleHide = useCallback(() => {
    if (onHide) onHide();
  }, [ onHide ]);

  return (
    <Modal
      footer={null}
      title="Best Checkpoint"
      visible={show}
      onCancel={handleHide}>
      <div className={css.base}>
        {checkpoint.uuid && <div data-label="UUID">{checkpoint.uuid}</div>}
        <div data-label="Experiment Id">
          <Badge>{checkpoint.experimentId}</Badge>
        </div>
        <div data-label="Trial Id">
          <Badge>{checkpoint.trialId}</Badge>
        </div>
        <div data-label="State">
          <Badge type={BadgeType.State}>{checkpoint.state}</Badge>
        </div>
        <div data-label="Location">{getStorageLocation(config, checkpoint)}</div>
        <div data-label="Validation Metric">{config.searcher.metric}</div>
        {checkpoint.validationMetric && <div data-label="Validation Value">
          {humanReadableFloat(checkpoint.validationMetric)}
        </div>}
        <div data-label="Start Time">{formatDatetime(checkpoint.startTime)}</div>
        {checkpoint.endTime && <div data-label="End Time">
          {formatDatetime(checkpoint.endTime)}
        </div>}
        <div data-label="Total Size">--</div>
        <div data-label="Resources">--</div>
      </div>
    </Modal>
  );
};

export default CheckpointModal;
