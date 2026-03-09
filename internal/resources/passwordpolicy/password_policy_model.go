package passwordpolicy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PasswordPolicyModel struct {
	ID                               types.String `tfsdk:"id"`
	DomainID                         types.String `tfsdk:"domain_id"`
	Name                             types.String `tfsdk:"name"`
	MinLength                        types.Int64  `tfsdk:"min_length"`
	MaxLength                        types.Int64  `tfsdk:"max_length"`
	MaxConsecutiveLetters            types.Int64  `tfsdk:"max_consecutive_letters"`
	ExpiryDuration                   types.Int64  `tfsdk:"expiry_duration"`
	OldPasswords                     types.Int64  `tfsdk:"old_passwords"`
	IncludeNumbers                   types.Bool   `tfsdk:"include_numbers"`
	IncludeSpecialCharacters         types.Bool   `tfsdk:"include_special_characters"`
	LettersInMixedCase               types.Bool   `tfsdk:"letters_in_mixed_case"`
	ExcludePasswordsInDictionary     types.Bool   `tfsdk:"exclude_passwords_in_dictionary"`
	ExcludeUserProfileInfoInPassword types.Bool   `tfsdk:"exclude_user_profile_info_in_password"`
	PasswordHistoryEnabled           types.Bool   `tfsdk:"password_history_enabled"`
	DefaultPolicy                    types.Bool   `tfsdk:"default_policy"`
}
