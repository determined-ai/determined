import { ExclamationCircleOutlined } from '@ant-design/icons';
import { ModalFuncProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Link from 'components/Link';
import { paths, routeToReactUrl } from 'routes/utils';
import { deleteExperiment } from 'services/api';
import { CheckpointDetail, CheckpointStorageType, CheckpointWorkload, CheckpointWorkloadExtended,
  ExperimentConfig, RunState } from 'types';
import { formatDatetime } from 'utils/datetime';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { humanReadableBytes } from 'utils/string';
import { checkpointSize, getBatchNumber } from 'utils/workload';

import useModal, { ModalHooks } from './useModal';
import css from './useModalCheckpoint.module.scss';

interface Props {
  checkpoint?: CheckpointWorkloadExtended | CheckpointDetail;
  config: ExperimentConfig;
  searcherValidation?: number;
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

const useModalCheckpoint = ({ checkpoint, config, title = 'checkpoint', ...props }: Props): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();

  const handleOk = useCallback(async () => {
    try {
      await deleteExperiment({ experimentId: 1 });
      routeToReactUrl(paths.experimentList());
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete experiment.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, []);

  const getContent = useCallback(() => {
    let content: any = <div>"no exp"</div>;
    if(checkpoint && checkpoint?.experimentId !== undefined && checkpoint?.resources !== undefined){
      const totalBatchesProcessed = getBatchNumber(checkpoint);
      const state = checkpoint.state as unknown as RunState;
      const totalSize = humanReadableBytes(checkpointSize(checkpoint));

      const searcherMetric = props.searcherValidation !== undefined ?
        props.searcherValidation :
        ('validationMetric' in checkpoint ? checkpoint.validationMetric : undefined);
      const checkpointResources = checkpoint.resources;
      const resources = Object.keys(checkpoint.resources)
        .sort((a, b) => checkpointResources[a] - checkpointResources[b])
        .map(key => ({ name: key, size: humanReadableBytes(checkpointResources[key]) }));
      content = (
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
    }
    return content;
  }, [ checkpoint, config, props.searcherValidation ]);

  const modalProps: ModalFuncProps = useMemo(() => {
    return {
      content: getContent(),
      icon: <ExclamationCircleOutlined />,
      okText: 'Delete',
      onOk: handleOk,
      title: 'Confirm Experiment Deletion',
    };
  }, [ handleOk, getContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalCheckpoint;
