import Button from 'hew/Button';
import { useTheme } from 'hew/Theme';
import { notification } from 'hew/Toast';
import { useState } from 'react';

import Link from 'components/Link';
import { paths } from 'routes/utils';
import { refreshPage } from 'utils/browser';

interface Props {
  version: string;
}

const VersionChecker: React.FC<Props> = ({ version }: Props) => {
  const {
    themeSettings: { className: themeClass },
  } = useTheme();
  const [closed, setClosed] = useState(false);
  // vi.stubEnv uses string type to set process.env.IS_DEV:
  const isDev = JSON.parse(process.env.IS_DEV ?? 'false');

  /*
   * Check to make sure the WebUI version matches the platform version.
   * Skip this check for development version.
   */
  if (!isDev && version !== process.env.VERSION) {
    const btn = (
      <Button type="primary" onClick={refreshPage}>
        Update Now
      </Button>
    );
    const message = 'New WebUI Version';
    const description = (
      <div>
        WebUI version <b>v{version}</b> is available. Check out what&apos;s new in our&nbsp;
        <Link external path={paths.docs('/release-notes.html')}>
          release notes
        </Link>
        .
      </div>
    );
    if (!closed) {
      setTimeout(() => {
        notification.warning({
          btn,
          className: themeClass,
          description,
          duration: 0,
          key: 'version-mismatch',
          message,
          onClose: () => setClosed(true),
          placement: 'bottomRight',
        });
      }, 0); // 0ms setTimeout needed to make sure UIProvider is available.
    }
  }

  return null;
};

export default VersionChecker;
