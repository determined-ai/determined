import yaml from 'js-yaml';
import React, { useEffect, useRef } from 'react';
import MonacoEditor from 'react-monaco-editor';

import Section from 'components/Section';
import useResize from 'hooks/useResize';
import { ExperimentBase } from 'types';

import css from './ExperimentConfiguration.module.scss';

interface Props {
  experiment: ExperimentBase;
}

const PADDING = 8;

const ExperimentConfiguration: React.FC<Props> = ({ experiment }: Props) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<MonacoEditor>(null);
  const resize = useResize(containerRef);

  useEffect(() => {
    editorRef.current?.editor?.layout();
  }, [ resize ]);

  return (
    <Section bodyNoPadding bodyScroll maxHeight>
      <div className={css.base} ref={containerRef}>
        <MonacoEditor
          height="100%"
          language="yaml"
          options={{
            minimap: { enabled: false },
            occurrencesHighlight: false,
            padding: { bottom: PADDING, top: PADDING },
            readOnly: true,
            scrollBeyondLastLine: false,
            selectOnLineNumbers: true,
          }}
          ref={editorRef}
          theme="vs-light"
          value={yaml.dump(experiment.configRaw)} />
      </div>
    </Section>
  );
};

export default ExperimentConfiguration;
