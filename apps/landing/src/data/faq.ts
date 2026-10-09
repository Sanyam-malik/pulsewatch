export type FaqItem = {
  question: string;
  answer: string;
};

export const faqItems: FaqItem[] = [
  {
    question: "What is Pulsewatch?",
    answer:
      "Pulsewatch is a community hard fork of Peekaping, an open-source, self-hosted uptime monitoring and status page tool built with Go and React.",
  },
  {
    question: "How does Pulsewatch compare to Uptime Kuma?",
    answer:
      "Pulsewatch continues Peekaping's API-first design and modular architecture, with a focus on extending the open-source monitoring stack.",
  },
  {
    question: "Does Pulsewatch have public status pages?",
    answer:
      "Yes. You can publish branded public status pages that show uptime, and performance metrics.",
  },
  {
    question: "How do I deploy Pulsewatch?",
    answer:
      "Use the Docker image coordinates currently published for the upstream project and the deployment instructions in this repository.",
  },
  {
    question: "Which databases are supported?",
    answer:
      "Pulsewatch retains the upstream project's MongoDB, PostgreSQL, and SQLite storage options.",
  },
  {
    question: "Is there a REST API?",
    answer:
      "Yes. Pulsewatch includes Swagger/OpenAPI documentation for automation and integrations.",
  },
  {
    question: "Can I migrate from Uptime Kuma?",
    answer:
      "A migration tool is being developed. For now, you can migrate manually.",
  },
  {
    question: "Is Pulsewatch free for commercial use?",
    answer:
      "Yes. It’s MIT-licensed and free for personal and commercial projects.",
  },
];

export default faqItems;

