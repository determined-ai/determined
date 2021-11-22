import { CopyOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import React, { useCallback, useState } from 'react';

type TextOptions = 'Copy to Clipboard' | 'Copied!'

const DEFAULT_TOOLTIP_TEXT: TextOptions = 'Copy to Clipboard';

interface Props {
  onCopy: () => Promise<void>;
}

const CopyButton: React.FC<Props> = ({ onCopy }: Props) => {
  const [ text, setText ] = useState<TextOptions>(DEFAULT_TOOLTIP_TEXT);

  const handleCopy = useCallback(async () => {
    await onCopy();
    setText('Copied!');
    setTimeout(() => {
      setText(DEFAULT_TOOLTIP_TEXT);
    }, 2000);
  }, [ onCopy ]);

  return (
    <Tooltip title={text}>
      <CopyOutlined onClick={handleCopy} />
    </Tooltip>
  );
};

export default CopyButton;
