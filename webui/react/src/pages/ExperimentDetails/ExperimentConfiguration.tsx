import yaml from 'js-yaml';
import React from 'react';

import Section from 'components/Section';
import Spinner from 'components/Spinner';
import { ExperimentBase } from 'types';

import css from './ExperimentConfiguration.module.scss';

interface Props {
  experiment: ExperimentBase;
}

const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

const ExperimentConfiguration: React.FC<Props> = ({ experiment }: Props) => {
  return (
    <Section bodyNoPadding bodyScroll maxHeight>
      <React.Suspense fallback={(
        <div className={css.loading}><Spinner tip="Loading text editor..." /></div>
      )}>
        <MonacoEditor
          height="100%"
          options={{
            occurrencesHighlight: false,
            readOnly: true,
          }}
          value={yaml.dump(experiment.configRaw)}
        />
      </React.Suspense>
    </Section>
  );
};

export default ExperimentConfiguration;
