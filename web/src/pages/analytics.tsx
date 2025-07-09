import { CONFIG } from 'src/config-global';

import { OverviewAnalyticsView } from 'src/sections/overview/view';

// ----------------------------------------------------------------------

export default function Page() {
  return (
    <>
      <title>{`Analytics - ${CONFIG.appName}`}</title>
      <meta
        name="description"
        content="Analytics dashboard for the application"
      />
      <meta name="keywords" content="application,dashboard,admin" />

      <OverviewAnalyticsView />
    </>
  );
}
