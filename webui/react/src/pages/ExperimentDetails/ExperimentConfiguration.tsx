import yaml from 'js-yaml';
import React from 'react';
import MonacoEditor from 'react-monaco-editor';

import Section from 'components/Section';
import { ExperimentBase } from 'types';

import css from './ExperimentConfiguration.module.scss';

interface Props {
  experiment: ExperimentBase;
}

const ExperimentConfiguration: React.FC<Props> = ({ experiment }: Props) => {
  return (
    <Section bodyBorder>
      <div className={css.base}>
        <MonacoEditor
          height="60vh"
          language="yaml"
          options={{
            minimap: { enabled: false },
            occurrencesHighlight: false,
            readOnly: true,
            scrollBeyondLastLine: false,
            selectOnLineNumbers: true,
          }}
          theme="vs-light"
          value={yaml.dump(experiment.configRaw)} />
      </div>
    </Section>
  );
};

export default ExperimentConfiguration;
