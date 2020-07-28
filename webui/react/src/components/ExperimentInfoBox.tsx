import { Button } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import Link from 'components/Link';
import { CheckpointDetail, CheckpointState, ExperimentDetails } from 'types';
import { formatDatetime } from 'utils/date';
import { floatToPercent, humanReadableFloat } from 'utils/string';

import CheckpointModal from './CheckpointModal';
import css from './ExperimentInfoBox.module.scss';
import ProgressBar from './ProgressBar';

interface Props {
  experiment: ExperimentDetails;
}

const renderRow = (label: string, value: React.ReactNode): React.ReactNode => {
  if (value === undefined) return <></>;
  return (
    <tr key={label}>
      <td className={css.label}>{label}</td>
      <td>
        {[ 'string', 'number' ].includes(typeof value) ?
          <span>{value}</span> : value
        }
      </td>
    </tr>
  );
};

const InfoBox: React.FC<Props> = ({ experiment: exp }: Props) => {
  const [ showBestCheckpoint, setShowBestCheckpoint ] = useState(false);

  const orderFactor = exp.config.searcher.smallerIsBetter ? 1 : -1;

  const bestValidation = useMemo(() => {
    const sortedValidations = exp.validationHistory
      .filter(a => a.validationError !== undefined)
      .sort((a, b) => (a.validationError as number - (b.validationError as number)) * orderFactor);
    return sortedValidations[0]?.validationError;
  }, [ exp.validationHistory, orderFactor ]);

  const bestCheckpoint: CheckpointDetail = useMemo(() => {
    const sortedCheckpoints: CheckpointDetail[] = exp.trials
      .filter(trial => trial.bestAvailableCheckpoint
        && trial.bestAvailableCheckpoint.validationMetric
        && trial.bestAvailableCheckpoint.state === CheckpointState.Completed)
      .map(trial => ({
        ...trial.bestAvailableCheckpoint,
        batch: trial.numBatches,
        experimentId: trial.experimentId,
        trialId: trial.id,
      }) as CheckpointDetail)
      .sort((a, b) => {
        return (a.validationMetric as number - (b.validationMetric as number)) * orderFactor;
      });
    return sortedCheckpoints[0];
  }, [ exp.trials, orderFactor ]);

  const handleShowBestCheckpoint = useCallback(() => setShowBestCheckpoint(true), []);
  const handleHideBestCheckpoint = useCallback(() => setShowBestCheckpoint(false), []);

  return (
    <div className={css.base}>
      <table>
        <tbody>
          {renderRow('State', <Badge state={exp.state} type={BadgeType.State} />)}
          {renderRow('Progress', exp.progress && <ProgressBar
            percent={exp.progress * 100}
            state={exp.state}
            title={floatToPercent(exp.progress, 0)} />)}
          {renderRow('Start Time', formatDatetime(exp.startTime))}
          {renderRow('End Time', exp.endTime && formatDatetime(exp.endTime))}
          {renderRow('Max Slot', exp.config.resources.maxSlots || 'Unlimited')}
          {bestValidation && renderRow(
            'Best Validation',
            `${humanReadableFloat(bestValidation)} (${exp.config.searcher.metric})`,
          )}
          {renderRow('Best Checkpoint', bestCheckpoint && (<>
            <Button onClick={handleShowBestCheckpoint}>
              Trial {bestCheckpoint.trialId} Batch {bestCheckpoint.batch}
            </Button>
            <CheckpointModal
              checkpoint={bestCheckpoint}
              config={exp.config}
              show={showBestCheckpoint}
              title={`Best Checkpoint for Experiment ${exp.id}`}
              onHide={handleHideBestCheckpoint} />
          </>))}
          {renderRow('Configuration',<Button disabled>Show</Button>)}
          {renderRow('Model Definition', <Button>
            <Link path={`/exps/${exp.id}/model_def`}>Download</Link>
          </Button>)}
        </tbody>
      </table>
    </div>
  );
};

export default InfoBox;
