import React from 'react';
import { Link } from 'react-router-dom';

import Icon from 'components/Icon';
import { paths } from 'routes/utils';
import { getStateColorCssVar } from 'themes';
import { ExperimentBase, TrialDetails } from 'types';

import css from './TrialHeaderLeft.module.scss';

interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialHeaderLeft: React.FC<Props> = ({ experiment, trial }: Props) => {
  return (
    <>
      <Link className={css.experiment} to={paths.experimentDetails(trial.experimentId)}>
        Experiment {trial.experimentId} | <span>{experiment.name}</span>
        <Icon name="arrow-right" size="tiny" />
      </Link>
      <div className={css.trial}>
        <div className={css.state} style={{ backgroundColor: getStateColorCssVar(trial.state) }}>
          {trial.state}
        </div>
        Trial {trial.id}
      </div>
    </>
  );
};

export default TrialHeaderLeft;
