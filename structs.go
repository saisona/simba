/**
 * File              : structs.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 14.11.2021
 * Last Modified Date: 14.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */
package simba

type SlackVerificationStruct struct {
	Type      string `json:"type"`
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
}
