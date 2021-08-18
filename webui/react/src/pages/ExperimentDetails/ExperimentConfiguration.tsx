import yaml from 'js-yaml';
import React, { useEffect, useRef } from 'react';
import MonacoEditor from 'react-monaco-editor';

import Section from 'components/Section';
import useResize from 'hooks/useResize';
import { ExperimentBase } from 'types';

interface Props {
  experiment: ExperimentBase;
}

const ExperimentConfiguration: React.FC<Props> = ({ experiment }: Props) => {
  const editor = useRef<MonacoEditor>(null);
  const resize = useResize();

  useEffect(() => {
    editor.current?.editor?.layout();
  }, [ resize ]);

  return (
    <Section bodyBorder bodyNoPadding bodyScroll maxHeight>
      <MonacoEditor
        height="100%"
        language="yaml"
        options={{
          minimap: { enabled: false },
          occurrencesHighlight: false,
          readOnly: true,
          scrollBeyondLastLine: false,
          selectOnLineNumbers: true,
        }}
        ref={editor}
        theme="vs-light"
        value={yaml.dump(experiment.configRaw)} />
    </Section>
  );
};

export default ExperimentConfiguration;
