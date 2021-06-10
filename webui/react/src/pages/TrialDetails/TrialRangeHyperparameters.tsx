import React, { useMemo } from 'react';

import { ExperimentBase, ExperimentHyperParam, TrialDetails } from 'types';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

interface HyperParameter {
  hyperparameter: string,
  value: ExperimentHyperParam,
}

const TrialRangeHyperparameters: React.FC<Props> = ({ experiment }: Props) => {
  const dataSource: HyperParameter[] = useMemo(() => {
    return Object.entries(experiment.config.hyperparameters).map(([ hyperparameter, value ]) => {
      return {
        hyperparameter,
        value,
      };
    });
  }, [ experiment.config.hyperparameters ]);

  return (
    <div>
      {dataSource.map(hp => <div key={hp.hyperparameter}>{JSON.stringify(hp.value)}</div>)}
    </div>
  );
};

export default TrialRangeHyperparameters;
