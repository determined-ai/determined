import { Loaded } from 'hew/utils/loadable';
import React from 'react';

import { TrialDetails } from 'types';
import handleError from 'utils/error';

import css from './Metadata.module.scss';
import Section from './Section';

const CodeEditor = React.lazy(() => import('hew/CodeEditor'));

interface Props {
  trial: TrialDetails;
}

const Metadata: React.FC<Props> = ({ trial }: Props) => {
  return (
    <div className={css.base}>
      <Section title="Metadata">
        <CodeEditor
          file={Loaded(JSON.stringify(trial.metadata, undefined, 2))}
          files={[{ key: `${trial.id}_metadata.json`, title: `${trial.id}_metadata.json` }]}
          readonly
          onError={handleError}
        />
      </Section>
    </div>
  );
};

export default Metadata;
