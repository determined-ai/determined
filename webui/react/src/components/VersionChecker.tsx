import Button from 'hew/Button';
import { useTheme } from 'hew/Theme';
import { notification } from 'hew/Toast';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import { useEffect } from 'react';

import Link from 'components/Link';
import { paths } from 'routes/utils';
import determinedStore from 'stores/determinedInfo';
import { refreshPage } from 'utils/browser';

const VersionChecker: React.FC = () => {
  const loadableInfo = useObservable(determinedStore.loadableInfo);

  const {
    themeSettings: { className: themeClass },
  } = useTheme();

  // need useEffect to avoid showing notification on each route change:
  useEffect(() => {
    /*
     * Check to make sure the WebUI version matches the platform version.
     * Skip this check for development version.
     */
    Loadable.quickMatch(loadableInfo, undefined, undefined, (info) => {
      if (!process.env.IS_DEV && info.version !== process.env.VERSION) {
        const btn = (
          <Button type="primary" onClick={refreshPage}>
            Update Now
          </Button>
        );
        const message = 'New WebUI Version';
        const description = (
          <div>
            WebUI version <b>v{info.version}</b> is available. Check out what&apos;s new in
            our&nbsp;
            <Link external path={paths.docs('/release-notes.html')}>
              release notes
            </Link>
            .
          </div>
        );
        setTimeout(() => {
          notification.warning({
            btn,
            className: themeClass,
            description,
            duration: 0,
            key: 'version-mismatch',
            message,
            placement: 'bottomRight',
          });
        }, 0); // need 0ms setTimeout to avoid UIProvider error
      }
    });
  }, [loadableInfo, themeClass]);

  return null;
};

export default VersionChecker;
