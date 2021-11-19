import { CopyOutlined } from '@ant-design/icons';
import { Input, notification, Popover, Tooltip } from 'antd';
import React, { PropsWithChildren, useCallback, useMemo, useState } from 'react';

import { ModelVersion } from 'types';
import { copyToClipboard } from 'utils/dom';

interface Props {
  modelVersion: ModelVersion;
}

const DownloadModelPopover: React.FC<Props> = (
  { children, modelVersion }: PropsWithChildren<Props>,
) => {
  const [ visible, setVisible ] = useState(false);

  const downloadCommand = useMemo(() => {
    return `det checkpoint download ${modelVersion.checkpoint.uuid}`;
  }, [ modelVersion.checkpoint.uuid ]);

  const handleCopy = useCallback(async () => {
    await copyToClipboard(downloadCommand);
    notification.open({ message: 'Copied to clipboard' });
    setVisible(false);
  }, [ downloadCommand ]);

  const handleVisibleChange = useCallback((visible) => {
    setVisible(visible);
  }, []);

  return (
    <Popover
      content={(
        <div>
          <Input
            suffix={(
              <Tooltip title="Copy to Clipboard">
                <CopyOutlined onClick={handleCopy} />
              </Tooltip>
            )}
            value={downloadCommand} />
          <p style={{ color: 'var(--theme-colors-monochrome-8)' }}>
            Copy/paste command into the Determined CLI
          </p>
        </div>
      )}
      placement="bottomRight"
      title="Download with Determined CLI"
      trigger="click"
      visible={visible}
      onVisibleChange={handleVisibleChange}>
      {children}
    </Popover>
  );
};

export default DownloadModelPopover;
