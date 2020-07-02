import { Button } from 'antd';
import { filter } from 'fp-ts/lib/ReadonlyRecord';
import React from 'react';

import Badge, { BadgeType } from 'components/Badge';
import Link from 'components/Link';
import { formatDatetime } from 'utils/date';
import { humanReadableFloat } from 'utils/string';

import { Checkpoint, CheckpointState, ExperimentDetails } from '../types';

import css from './ExperimentInfoBox.module.scss';

interface Props {
  experiment: ExperimentDetails;
}

// const pair = (label: string, value: React.ReactNode | string): React.ReactNode => {
//   return (
//     <div className={css.pair}>
//       <span className={css.label}>{label}</span>
//       {typeof value === 'string' ?
//         <span>{value}</span> :
//         { value }
//       }
//     </div>
//   );
// };

const pairRow = (label: string, value: React.ReactNode | undefined): React.ReactNode => {
  return (
    <tr key={label}>
      <td className={css.label}>{label}</td>
      <td>
        {[ 'string', 'number' ].includes(typeof value) ?
          <span>{value}</span> :
          <>
            { value }
          </>
        }
      </td>
    </tr>
  );
};

const InfoBox: React.FC<Props> = ({ experiment: exp }: Props) => {
  // CHECK orderfactor
  const orderFator = exp.config.searcher.smallerIsBetter ? 1 : -1;
  const sortedValidations = exp.validationHistory
    .filter(a => a.validationError !== undefined)
    .sort((a, b) => (a.validationError as number - (b.validationError as number)) * orderFator);

  const bestVal = sortedValidations[0]?.validationError;

  const sortedCheckpoints: Checkpoint[] = exp.trials
    .filter(trial => trial.bestAvailableCheckpoint
      && trial.bestAvailableCheckpoint.validationMetric
      && trial.bestAvailableCheckpoint.state === CheckpointState.Completed)
    .map(trial => trial.bestAvailableCheckpoint as Checkpoint)
    .sort((a, b) => (a.validationMetric as number - (b.validationMetric as number)) * orderFator);

  // best available checkpoint.
  const bestCheckpoint = sortedCheckpoints[0];

  // const infoBox: Record<string, React.ReactNode> = {
  //   'Best Checkpoint': <Button type="primary">Trial 1 batch 700</Button>,
  //   'Start Time': props.startTime,
  //   'State': props.state,
  // };

  // const infoBox2: [string, Element] = [
  //   [ 'Best Checkpoint', <Button type="primary">Trial 1 batch 700</Button> ],
  // 'Start Time': props.startTime,
  // 'State': props.state,
  // ];

  return (
    <div className={css.base}>
      {/* <table>
        {Object.entries(infoBox).map(([ label, value ]) => pairRow(label, value))}
      </table> */}
      <table>
        <tbody>
          {pairRow('State', <Badge state={exp.state} type={BadgeType.State} />)}
          {pairRow('Progress', exp.progress)}
          {pairRow('Start Time', formatDatetime(exp.startTime))}
          {pairRow('End Time', exp.endTime && formatDatetime(exp.endTime))}
          {pairRow('Max Slot', exp.config.resources.maxSlots || 'Unlimited')}
          {pairRow(`Best Validation (${exp.config.searcher.metric})`,
            bestVal && humanReadableFloat(bestVal))}
          {pairRow('Best Checkpoint', bestCheckpoint && <Button type="primary">
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
