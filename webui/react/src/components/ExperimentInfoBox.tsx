import { Button } from 'antd';
import React from 'react';

import Badge, { BadgeType } from 'components/Badge';
import Link from 'components/Link';
import { formatDatetime } from 'utils/date';
import { floatToPercent, humanReadableFloat } from 'utils/string';

import { Checkpoint, CheckpointState, ExperimentDetails } from '../types';

import css from './ExperimentInfoBox.module.scss';
import ProgressBar from './ProgressBar';

interface Props {
  experiment: ExperimentDetails;
}

const pairRow = (label: string, value: React.ReactNode): React.ReactNode => {
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
  const orderFactor = exp.config.searcher.smallerIsBetter ? 1 : -1;
  const sortedValidations = exp.validationHistory
    .filter(a => a.validationError !== undefined)
    .sort((a, b) => (a.validationError as number - (b.validationError as number)) * orderFactor);

  const bestVal = sortedValidations[0]?.validationError;

  const sortedCheckpoints: Checkpoint[] = exp.trials
    .filter(trial => trial.bestAvailableCheckpoint
      && trial.bestAvailableCheckpoint.validationMetric
      && trial.bestAvailableCheckpoint.state === CheckpointState.Completed)
    .map(trial => trial.bestAvailableCheckpoint as Checkpoint)
    .sort((a, b) => (a.validationMetric as number - (b.validationMetric as number)) * orderFactor);

  // best available checkpoint.
  const bestCheckpoint = sortedCheckpoints[0];

  return (
    <div className={css.base}>
      <table>
        <tbody>
          {pairRow('State', <Badge state={exp.state} type={BadgeType.State} />)}
          {pairRow('Progress', exp.progress && <ProgressBar
            percent={exp.progress * 100}
            state={exp.state}
            title={floatToPercent(exp.progress, 0)} />)}
          {pairRow('Start Time', formatDatetime(exp.startTime))}
          {pairRow('End Time', exp.endTime && formatDatetime(exp.endTime))}
          {pairRow('Max Slot', exp.config.resources.maxSlots || 'Unlimited')}
          {pairRow(`Best Validation (${exp.config.searcher.metric})`,
            bestVal && humanReadableFloat(bestVal))}
          {pairRow('Best Checkpoint', bestCheckpoint && <Button disabled type="primary">
            Trial {bestCheckpoint.trialId}
          </Button> )}
          {pairRow('Configuration',<Button disabled type="primary">Show</Button>)}
          {pairRow('Model Definition', <Button type="primary">
            <Link path={`/exps/${exp.id}/model_def`}>Download</Link>
          </Button>)}
        </tbody>
      </table>
    </div>
  );
};

export default InfoBox;
