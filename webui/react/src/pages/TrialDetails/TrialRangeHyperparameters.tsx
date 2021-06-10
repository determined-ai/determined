import React, { useMemo } from 'react';

import HumanReadableFloat from 'components/HumanReadableFloat';
import {
  ExperimentBase, ExperimentHyperParam, ExperimentHyperParamType,
  RawJson, TrialDetails, TrialHyperParameters,
} from 'types';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

interface HyperParameter {
  name: string,
  value: ExperimentHyperParam,
}

const TrialRangeHyperparameters: React.FC<Props> = ({ experiment, trial }: Props) => {
  const configSource: HyperParameter[] = useMemo(() => {
    return Object.entries(experiment.config.hyperparameters).map(([ name, value ]) => {
      return {
        name,
        value,
      };
    });
  }, [ experiment.config.hyperparameters ]);

  const valueSource: {name: string; value: number | string | boolean | RawJson}[] = useMemo(() => {
    return Object.entries(trial.hparams).map(([ name, value ]) => {
      return {
        name,
        value,
      };
    });
  }, [ trial.hparams ]);

  return (
    <div style={{ display: 'flex', gap: 20 }}>
      {configSource.map(hp => <div key={hp.name}>
        <HyperparameterRange
          config={hp}
          value={valueSource.find(hparam => hparam.name === hp.name) || { name: '', value: '' }} />
      </div>)}
    </div>
  );
};

interface RangeProps {
  config: HyperParameter
  value: TrialHyperParameters
}

const HyperparameterRange:React.FC<RangeProps> = ({ config, value }: RangeProps) => {

  return (
    <div style={{ alignSelf: 'center', display: 'flex', flexDirection: 'column' }}>
      {config.name}
      <div style={{ display: 'flex', height: 300, justifyContent: 'center', width: '100%' }}>
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          textAlign: 'right',
        }}>
          <p style={{ margin: 0 }}>{config.value.maxval}</p>
          <p style={{ margin: 0 }}>{config.value.minval}</p>
        </div>
        <div
          style={{
            alignSelf: 'center',
            backgroundColor: 'lightgray',
            borderRadius: 5,
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            justifyContent: 'space-between',
            marginInline: 5,
            width: 10,
          }}>
          {config.value.vals?.map(op =>
            <div
              key={op.toString()}
              style={{
                backgroundColor: 'blue',
                borderRadius: '100%',
                height: 12,
                left: -12/8,
                position: 'relative',
                width: 12,
              }}
            />)}
        </div>
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          height: `${100}%`,
          justifyContent: 'end',
        }}>
          <div style={{ display: 'flex' }}>
            <div style={{
              borderColor: 'transparent lightgray transparent',
              borderStyle: 'solid',
              borderWidth: '20px 20px 20px 0',
            }} />
            <div style={{ backgroundColor: 'lightgray', padding: 8, paddingBottom: 0 }}>
              <ParsedHumanReadableValue hp={value} type={config.value.type} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

const ParsedHumanReadableValue = (hp: TrialHyperParameters, type: ExperimentHyperParamType) => {
  switch (type) {
    case ExperimentHyperParamType.Categorical:
      return <p style={{ margin: 0 }}>{hp.value}</p>;
    case ExperimentHyperParamType.Constant:
      return <p style={{ margin: 0 }}>{hp.value}</p>;
    case ExperimentHyperParamType.Double:
      return <HumanReadableFloat num={parseFloat(hp.value as string)} />;
    case ExperimentHyperParamType.Int:
      return <p style={{ margin: 0 }}>{parseInt(hp.value as string)}</p>;
    case ExperimentHyperParamType.Log:
      return <HumanReadableFloat num={parseFloat(hp.value as string)} />;
    default:
      return <p style={{ margin: 0 }}>Err</p>;
  }
};

export default TrialRangeHyperparameters;
