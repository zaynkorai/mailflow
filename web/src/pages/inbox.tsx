import { CONFIG } from 'src/config-global';

import { InboxView } from 'src/sections/user/view';

// ----------------------------------------------------------------------

export default function Page() {
  return (
    <>
      <title>{`Inbox - ${CONFIG.appName}`}</title>

      <InboxView />
    </>
  );
}
