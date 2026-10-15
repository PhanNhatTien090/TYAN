// English rendering of the upstream sign-in notice. Original: README.zh-CN.md.
export const usageTerms = [
  {
    title: "Definitions and open-source license",
    body: "TYAN is derived from ARTEX and is released under the GNU Affero General Public License v3.0 (AGPL-3.0). You may use, copy, modify, and distribute the software under that license. Derivative works, including modified versions offered to users over a network, must comply with AGPL-3.0 and provide the corresponding complete source code. The bundled LICENSE contains the full license terms.",
  },
  {
    title: "Permitted scope of use",
    body: "The upstream usage notice permits personal learning, source-code research, study of security principles, and technical verification in a local isolated environment that you set up yourself. Permitted purposes include learning, academic research, and code review that are non-aggressive and non-destructive. Uses beyond those expressly permitted are prohibited by the upstream notice.",
  },
  {
    title: "Prohibited conduct",
    body: "The upstream notice prohibits the following:",
    items: [
      "Scanning, probing, exploiting, or attacking any website, online service, or networked system belonging to others or third parties, regardless of authorization or whether you own the assets.",
      "Using the software for actual penetration testing, offensive/defensive operations, red/blue team exercises, or production environments.",
      "Illegal intrusion, data theft, extortion, denial of service (DoS/DDoS), or any destructive or criminal activity.",
      "Removing, altering, or circumventing copyright, license, or security notices in the software or its output.",
      "Activities that violate applicable laws, regulations, or regulatory requirements in your country or region.",
    ],
  },
  {
    title: "Intellectual property",
    body: "Copyright and related intellectual property rights belong to the project authors and contributors. Rights are granted only within the scope of AGPL-3.0. This notice does not grant any other rights, expressly or implicitly.",
  },
  {
    title: "Data and privacy",
    body: "The upstream notice describes the software as self-hosted and states that its authors do not operate a centralized service or collect or upload your usage data. You control and are responsible for the lawfulness and security of all data you generate, process, or access, and bear the consequences of improper handling. Connections to LLM providers and other services are determined by your configuration.",
  },
  {
    title: "Compliance and legal responsibility",
    body: "You must comply with all applicable cybersecurity, data security, personal information protection, and computer crime laws in your country or region. The upstream notice specifically refers to mainland China's Cybersecurity Law, Data Security Law, Personal Information Protection Law, and relevant judicial interpretations. You alone bear legal liability and consequences arising from violations of applicable law or this notice, independently of the authors and contributors.",
  },
  {
    title: "Disclaimer and limitation of liability",
    body: "The software is provided AS IS and AS AVAILABLE, without express or implied warranties, including merchantability, fitness for a particular purpose, accuracy, or non-infringement. To the maximum extent permitted by law, the authors and contributors are not liable for direct, indirect, incidental, special, or consequential losses arising from use or inability to use the software, whether used properly or not, including data loss, system damage, business interruption, lost profits, or legal disputes.",
  },
  {
    title: "Changes and interpretation",
    body: "The upstream authors reserve the right to update the notice to reflect laws or project development. Updates are released with the project and take effect on publication; continued use constitutes acceptance of revised terms. To the extent permitted by law, final interpretation rests with the project authors. Invalidity of one provision does not affect the remaining provisions. Downloading, installing, accessing, or using the software indicates that you have read, understood, and accepted this notice.",
  },
] as const;
