import yaml from 'js-yaml';
import React from 'react';

import Section from 'components/Section';
import Spinner from 'shared/components/Spinner/Spinner';
import { ExperimentBase } from 'types';

import css from './ExperimentConfiguration.module.scss';

interface Props {
  experiment: ExperimentBase;
}

const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

const ExperimentConfiguration: React.FC<Props> = ({ experiment }: Props) => {
  /**
   * strip registry_auth from config for display
   * as well as workspace/project names
   */
  let publicConfig = {};
  if (experiment.configRaw) {
    const {
      environment: { registry_auth, ...restEnvironment },
      workspace,
      project,
      ...restConfig
    } = experiment.configRaw;
    publicConfig = { environment: restEnvironment, ...restConfig };
  }

  return (
    <Section bodyNoPadding bodyScroll maxHeight>
      <React.Suspense
        fallback={
          <div className={css.loading}>
            <Spinner tip="Loading text editor..." />
          </div>
        }>
        <MonacoEditor
          height="100%"
          options={{
            occurrencesHighlight: false,
            readOnly: true,
          }}
          value={yaml.dump(publicConfig)}
        />
      </React.Suspense>
    </Section>
  );
};

export default ExperimentConfiguration;
