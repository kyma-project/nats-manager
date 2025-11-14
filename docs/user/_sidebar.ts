export default [
  { text: 'Accessing the NATS Server Using CLI', link: './01-10-access-nats-server' },
  { text: 'NATS Custom Resource', link: './01-05-nats-custom-resource' },
  { text: 'Troubleshooting', link: './troubleshooting/README.md', collapsed: true, items: [
    { text: 'General Diagnostics: NATS Module Readiness and Connectivity', link: './troubleshooting/03-05-nats-troubleshooting' },
    { text: 'Published Events Are Pending in the Stream', link: './troubleshooting/03-10-fix-pending-events' }
    ] },
  ]
