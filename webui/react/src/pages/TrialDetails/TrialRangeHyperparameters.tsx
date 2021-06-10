import React, { useMemo } from 'react';

import { ExperimentBase, ExperimentHyperParam, TrialDetails } from 'types';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

interface HyperParameter {
  name: string,
  value: ExperimentHyperParam,
}

const TrialRangeHyperparameters: React.FC<Props> = ({ experiment }: Props) => {
  const dataSource: HyperParameter[] = useMemo(() => {
    return Object.entries(experiment.config.hyperparameters).map(([ name, value ]) => {
      return {
        name,
        value,
      };
    });
  }, [ experiment.config.hyperparameters ]);

  return (
    <div style={{ display: 'flex', gap: 20 }}>
      {dataSource.map(hp => <div key={hp.name}>
        <HyperparameterRange name={hp.name} value={hp.value} />
      </div>)}
    </div>
  );
};

const HyperparameterRange:React.FC<HyperParameter> = ({ name, value }: HyperParameter) => {
  return <div style={{ display: 'flex', flexDirection: 'column' }}>{name}<div
    style={{
      alignSelf: 'center',
      backgroundColor: 'lightgray',
      borderRadius: 5,
      display: 'flex',
      flexDirection: 'column',
      height: 200,
      justifyContent: 'space-between',
      width: 10,
    }}>
    {value.vals?.map(op =>
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
  </div></div>;
};

export default TrialRangeHyperparameters;
