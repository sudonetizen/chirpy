package auth

import (
    "time"
    "testing"
    "github.com/google/uuid"
)

func TestMakeJWTAndValidateJWT(t *testing.T) {
    tests := []struct {
        userID uuid.UUID
        tokenS string
        expire time.Duration
    }{
        {
            userID: uuid.New(),
            tokenS: "ThisIsSecret",
            expire: time.Duration(20 * time.Second),
        },        
        {
            userID: uuid.New(),
            tokenS: "TheTopSecret",
            expire: time.Duration(20 * time.Second), 
        },        
    }

    for _, tst := range tests {
        tkn, err := MakeJWT(tst.userID, tst.tokenS, tst.expire)
        if err != nil {t.Errorf("error with MakeJWT: %v\n", err)}

        pip, err := ValidateJWT(tkn, tst.tokenS)
        if err != nil {t.Errorf("error with ValidateJWT: %v\n", err)}

        if tst.userID != pip {t.Errorf("not equal")}
    }
}
